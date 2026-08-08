package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"virtualnet/common"
)

// bufferedConn 包装 net.Conn，在读取时先消费 json.Decoder 残留的缓冲数据，
// 避免 io.Copy 直接读 conn 时跳过被 Decoder 多读的字节，导致数据流错位。
type bufferedConn struct {
	net.Conn
	reader io.Reader
}

func (bc *bufferedConn) Read(p []byte) (int, error) {
	return bc.reader.Read(p)
}

// 处理TCP连接（认证+转发）
func handleTCPConnection(conn net.Conn) {
	needClose := true
	defer func() {
		if needClose {
			conn.Close()
		}
	}()

	// 读取认证信息
	var auth common.TCPAuth
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&auth); err != nil {
		log.Printf("TCP认证失败（解析错误）: %v", err)
		return
	}

	// 将 Decoder 可能多读的缓冲数据拼回连接，防止 io.Copy 丢失首包字节导致数据流错位
	conn = &bufferedConn{Conn: conn, reader: io.MultiReader(decoder.Buffered(), conn)}

	log.Printf("TCP 认证信息：%+v", auth)

	// 验证令牌有效性（5分钟内有效）
	mutex.RLock()
	srcClient, srcExists := clients[auth.VirtualIP]
	mutex.RUnlock()

	if !srcExists || srcClient.token != auth.Token || time.Since(auth.Timestamp) > 5*time.Minute {
		log.Printf("TCP认证失败（无效令牌）: %s -> %s", auth.VirtualIP, auth.ConnectFor)
		return
	}

	// 验证目标客户端是否存在
	mutex.RLock()
	dstClient, dstExists := clients[auth.ConnectFor]
	mutex.RUnlock()

	if !dstExists {
		log.Printf("TCP连接失败（目标不存在）: %s -> %s", auth.VirtualIP, auth.ConnectFor)
		sendWSMessage(srcClient, common.MsgTypeError, errorContent(404, "目标节点不存在"))
		return
	}

	// 存储TCP连接（源客户端 -> 目标客户端）
	srcClient.mu.Lock()
	srcClient.tcpConns[auth.ConnectFor] = conn
	srcClient.mu.Unlock()

	dstClient.mu.Lock()
	_, exists := dstClient.tcpConns[auth.VirtualIP]
	dstClient.mu.Unlock()

	if exists {
		log.Printf("%s 已有TCP连接等待", auth.VirtualIP)
		needClose = false
		return
	}

	log.Printf("TCP连接建立: %s -> %s", auth.VirtualIP, auth.ConnectFor)

	// 等待目标客户端的反向连接
	dstConn := waitForReverseConn(dstClient, auth.VirtualIP)
	if dstConn == nil {
		log.Printf("TCP连接失败（无反向连接）: %s <-> %s", auth.VirtualIP, auth.ConnectFor)
		srcClient.mu.Lock()
		delete(srcClient.tcpConns, auth.ConnectFor)
		srcClient.mu.Unlock()
		return
	}

	// 通知双方认证成功
	sendWSMessage(srcClient, common.MsgTypeAuthSuccess, []byte(`{"message": "与 `+auth.ConnectFor+` 连接成功"}`))
	sendWSMessage(dstClient, common.MsgTypeAuthSuccess, []byte(`{"message": "与 `+auth.VirtualIP+` 连接成功"}`))

	// 双向转发流量
	go func() {
		io.Copy(conn, dstConn)
		cleanupConns(srcClient, auth.ConnectFor, dstClient, auth.VirtualIP)
	}()
	io.Copy(dstConn, conn)
	cleanupConns(srcClient, auth.ConnectFor, dstClient, auth.VirtualIP)
}

// 等待目标客户端的反向TCP连接（最多等5秒）
func waitForReverseConn(dstClient *Client, srcIP string) net.Conn {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil
		case <-ticker.C:
			dstClient.mu.Lock()
			conn, exists := dstClient.tcpConns[srcIP]
			dstClient.mu.Unlock()
			if exists {
				return conn
			}
		}
	}
}
