package client

import (
	"flag"
	"log"
	"net"
	"sync"

	"virtualnet/common"

	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/tun"
)

// Client 表示虚拟网络中的客户端
type Client struct {
	Name      string `json:"name"`
	conn      net.Conn
	Latency   int    `json:"latency"`
	VirtualIp string `json:"virtualIp"`
	mu        sync.Mutex
}

func (c *Client) SetId(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Name = id
}

func (c *Client) SetLatency(latency int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Latency = latency
}

func getCurClient() *Client {
	return tcpConns[virtualIP]
}

var (
	// serverWS 仅注册 flag，默认值在 Run() 内结合 .env / 环境变量动态求值，
	// 避免 .env 还未加载时包级初始化就把默认值算死。
	serverWS = flag.String("server", "", "WebSocket服务器地址（留空则读 VIRNET_SERVER 环境变量）")
	virtualIP string             // 本地虚拟IP（服务端下发）
	serverTCP string             // 服务器TCP地址
	token     string             // 认证令牌
	wsConn    *websocket.Conn    // WebSocket连接
	tcpConns  map[string]*Client // 按目标IP存储的TCP连接（key: 目标虚拟IP）
	mutex     sync.Mutex         // 保护共享变量
	tunDevice tun.Device         // Wintun虚拟网卡设备
)

// 客户端状态变更回调函数
var onClientChangeFunc []func()

// resolveServerWS 按优先级确定最终连接的服务器地址：
// 命令行 -server > 环境变量 VIRNET_SERVER（含 .env）> 默认硬编码。
func resolveServerWS() string {
	if *serverWS != "" {
		return *serverWS
	}
	return common.Getenv("VIRNET_SERVER", "ws://43.138.247.132:8080/ws")
}

// Run 启动客户端
func Run() {
	tcpConns = make(map[string]*Client)

	serverAddr := resolveServerWS()
	log.Printf("连接服务器: %s", serverAddr)

	// 连接WebSocket服务器
	var err error
	wsConn, _, err = websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatalf("连接WS服务器失败: %v", err)
	}
	defer wsConn.Close()

	log.Println("已连接到服务器，等待分配虚拟IP...")

	// 启动WS消息处理
	go handleWS()

	select {}
}

func SetName(name string) error {
	client := getCurClient()

	client.SetId(name)
	return sendRegisterMsg(name)
}

// GetClients 获取当前连接的客户端列表
func GetClients() []*Client {
	mutex.Lock()
	defer mutex.Unlock()

	curClient := getCurClient()
	var clients []*Client
	for _, client := range tcpConns {
		addClient := &Client{
			Name:      client.Name,
			VirtualIp: client.VirtualIp,
			Latency:   client.Latency,
		}
		// 本机放第一个
		if client == curClient {
			clients = append([]*Client{addClient}, clients...)
		} else {
			clients = append(clients, addClient)
		}
	}

	return clients
}

// OnClientChange 注册客户端状态变更回调
func OnClientChange(fc func()) {
	onClientChangeFunc = append(onClientChangeFunc, fc)
}

// notifyClientChange 通知客户端状态变更
func notifyClientChange() {
	for _, f := range onClientChangeFunc {
		f()
	}
}
