package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
