package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"virtualnet/common"
)

// Client 表示虚拟网络中的客户端
type Client struct {
	name                 string              // 展示的昵称
	virtualIP            string              // 分配的虚拟IP
	wsConn               *websocket.Conn     // WebSocket连接
	tcpConns             map[string]net.Conn // 按目标IP存储的TCP连接（key: 目标虚拟IP）
	token                string              // 认证令牌
	latency              int64               // 延迟
	lastLatencyStartTime time.Time           // 延迟测试起始时间
	mu                   sync.Mutex          // 保护tcpConns的并发安全
	close                chan struct{}
}

func (c *Client) setName(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.name = name
}

var (
	clients = make(map[string]*Client) // 虚拟IP -> 客户端
	ipPool  = make(chan string, 253)   // 192.168.32.2-254（共253个IP）
	mutex   sync.RWMutex               // 保护clients的并发安全
)

// 初始化IP池
func initIPPool() {
	for i := 2; i <= 254; i++ {
		ipPool <- fmt.Sprintf("192.168.32.%d", i)
	}
}

// 生成认证令牌
func generateToken(virtualIP string) string {
	return fmt.Sprintf("%s-%d", virtualIP, time.Now().UnixNano())
}

// 向新客户端发送连接指令（连接所有已在线客户端），并向所有已在线客户端发送连接指令（连接新客户端）
func triggerConnections(newClient *Client) {
	mutex.RLock()
	defer mutex.RUnlock()

	// 遍历所有已在线的客户端（排除新客户端自己）
	for existingIP, existingClient := range clients {
		if existingIP == newClient.virtualIP {
			continue // 跳过自己
		}

		// 检查新客户端与现有客户端是否已建立连接（避免重复）
		newClient.mu.Lock()
		_, newHasConn := newClient.tcpConns[existingIP]
		newClient.mu.Unlock()

		existingClient.mu.Lock()
		_, existingHasConn := existingClient.tcpConns[newClient.virtualIP]
		existingClient.mu.Unlock()

		if newHasConn || existingHasConn {
			continue // 已存在连接，跳过
		}

		// 1. 通知新客户端连接现有客户端
		connectContent := common.ConnectContent{
			ConnectFor: existingIP,
			FromIP:     newClient.virtualIP,
		}
		content, _ := json.Marshal(connectContent)
		sendWSMessage(newClient, common.MsgTypeConnect, content)

		// 2. 通知现有客户端连接新客户端
		existingConnectContent := common.ConnectContent{
			ConnectFor: newClient.virtualIP,
			FromIP:     existingIP,
		}
		existingContent, _ := json.Marshal(existingConnectContent)
		sendWSMessage(existingClient, common.MsgTypeConnect, existingContent)

		log.Printf("触发连接: %s <-> %s", newClient.virtualIP, existingIP)
	}
}

// 清理连接
func cleanupConns(src *Client, dstIP string, dst *Client, srcIP string) {
	src.mu.Lock()
	if conn, ok := src.tcpConns[dstIP]; ok {
		conn.Close()
		delete(src.tcpConns, dstIP)
	}
	src.mu.Unlock()

	dst.mu.Lock()
	if conn, ok := dst.tcpConns[srcIP]; ok {
		conn.Close()
		delete(dst.tcpConns, srcIP)
	}
	dst.mu.Unlock()

	log.Printf("TCP连接断开: %s <-> %s", srcIP, dstIP)
}
