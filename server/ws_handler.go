package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"virtualnet/common"
)

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	wsMux    sync.Mutex // websocket串行使用
)

// 处理WebSocket连接
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 分配虚拟IP
	virtualIP, ok := <-ipPool
	if !ok {
		conn.WriteMessage(websocket.TextMessage, errorMessage(500, "IP池耗尽"))
		return
	}

	// 创建客户端实例
	token := generateToken(virtualIP)
	client := &Client{
		virtualIP: virtualIP,
		wsConn:    conn,
		tcpConns:  make(map[string]net.Conn),
		token:     token,
		close:     make(chan struct{}),
	}
	defer func() {
		close(client.close)
	}()

	// 注册客户端
	mutex.Lock()
	clients[virtualIP] = client
	mutex.Unlock()

	log.Printf("客户端上线: %s", virtualIP)

	// 发送分配IP消息
	assignContent := common.AssignIPContent{
		VirtualIP:  virtualIP,
		ServerTCP:  tcpAddr,
		ServerWS:   wsAddr,
		ExpireTime: time.Now().Add(24 * time.Hour).Unix(),
		Token:      token,
	}
	assignData, _ := json.Marshal(assignContent)
	sendWSMessage(client, common.MsgTypeAssignIP, assignData)

	// 触发与所有已在线客户端的连接
	triggerConnections(client)

	// 启动延迟测试定时器
	go startLatencyTicker(client)

	// 处理客户端消息
	handleClientMessages(client)

	// 清理客户端
	cleanupClient(client, virtualIP)
}

// 启动延迟测试定时器
func startLatencyTicker(client *Client) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		client.mu.Lock()
		client.lastLatencyStartTime = time.Now()
		client.mu.Unlock()
		_ = sendWSMessage(client, common.MsgTypePing, nil)

		select {
		case <-client.close:
			return
		case <-ticker.C:
		}
	}
}

// 处理客户端消息
func handleClientMessages(client *Client) {
	for {
		_, data, err := client.wsConn.ReadMessage()
		if err != nil {
			log.Printf("客户端 %s 断开连接: %v", client.virtualIP, err)
			break
		}

		var msg common.WSMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			log.Printf("客户端WS回复无法反序列化消息: %v", err)
			continue
		}

		handleClientWS(client, msg)
	}
}

// 清理客户端资源
func cleanupClient(client *Client, virtualIP string) {
	// 从客户端列表移除
	mutex.Lock()
	delete(clients, virtualIP)
	ipPool <- virtualIP // 回收IP
	mutex.Unlock()

	// 关闭所有TCP连接
	client.mu.Lock()
	for _, tcpConn := range client.tcpConns {
		tcpConn.Close()
	}
	client.tcpConns = nil
	client.mu.Unlock()

	log.Printf("客户端下线: %s", virtualIP)
}

// 处理客户端WebSocket消息
func handleClientWS(client *Client, msg common.WSMessage) {
	switch msg.Type {
	case common.MsgTypePong:
		client.mu.Lock()
		client.latency = time.Since(client.lastLatencyStartTime).Milliseconds()
		log.Printf("延迟测试: %dms", client.latency)
		defer client.mu.Unlock()

	case common.MsgRegisterName:
		var registerInfo common.RegisterInfoContent
		err := json.Unmarshal(msg.Content, &registerInfo)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		client.setName(registerInfo.Name)

		broadcastClientInfo()
	}
}
