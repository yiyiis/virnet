package client

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"golang.zx2c4.com/wireguard/tun"
	"virtualnet/common"
)

// Client 表示虚拟网络中的客户端
type Client struct {
	Name      string `json:"name"`
	conn      net.Conn
	Latency   int    `json:"latency"`
	VirtualIp string `json:"virtualIp"`
	mu        sync.Mutex
}

// PeerInfo 表示一个在线节点信息（供前端展示全部在线节点）
type PeerInfo struct {
	Name      string `json:"name"`
	VirtualIP string `json:"virtualIp"`
	Latency   int64  `json:"latency"`
	Connected bool   `json:"connected"`
	IsLocal   bool   `json:"isLocal"`
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
	serverWS   = flag.String("server", "ws://8.138.243.253:8080/ws", "WebSocket服务器地址")
	virtualIP  string             // 本地虚拟IP（服务端下发）
	serverTCP  string             // 服务器TCP地址
	token      string             // 认证令牌
	wsConn     *websocket.Conn    // WebSocket连接
	tcpConns   map[string]*Client // 按目标IP存储的TCP连接（key: 目标虚拟IP）
	mutex      sync.Mutex         // 保护共享变量
	tunDevice  tun.Device         // Wintun虚拟网卡设备
	onlinePeers []PeerInfo        // 全部在线节点（来自服务器广播，独立于tcpConns）
)

// 客户端状态变更回调函数
var onClientChangeFunc []func()

// Run 启动客户端
func Run() {
	tcpConns = make(map[string]*Client)

	// 连接WebSocket服务器
	var err error
	wsConn, _, err = websocket.DefaultDialer.Dial(*serverWS, nil)
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

// ConnectTo 主动向目标虚拟IP发起连接（供前端调用）
func ConnectTo(targetIP string) error {
	if targetIP == "" {
		return errors.New("目标IP不能为空")
	}

	mutex.Lock()
	localIP := virtualIP
	_, exists := tcpConns[targetIP]
	mutex.Unlock()

	if targetIP == localIP {
		return errors.New("不能连接自己")
	}
	if exists {
		return errors.New("已与该节点建立连接")
	}

	// 通过WS向服务器发送连接请求，由服务器通知被动方建立反向TCP连接
	content, err := json.Marshal(common.ConnectContent{
		ConnectFor: targetIP,
		FromIP:     localIP,
	})
	if err != nil {
		return err
	}
	data, err := json.Marshal(common.WSMessage{
		Type:    common.MsgTypeConnectReq,
		Content: content,
	})
	if err != nil {
		return err
	}
	if err := sendWSMessage(data); err != nil {
		return err
	}

	// 本地同时发起TCP拨号（与被动方的反向连接在服务器汇合）
	// TODO: 私有态需被动方同意，当前默认公开自动连接
	go establishTCPConnection(targetIP)
	return nil
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

// GetOnlinePeers 获取全部在线节点列表（含连接状态），供前端展示
func GetOnlinePeers() []PeerInfo {
	mutex.Lock()
	defer mutex.Unlock()

	peers := make([]PeerInfo, len(onlinePeers))
	for i, p := range onlinePeers {
		_, connected := tcpConns[p.VirtualIP]
		peers[i] = PeerInfo{
			Name:      p.Name,
			VirtualIP: p.VirtualIP,
			Latency:   p.Latency,
			Connected: connected,
			IsLocal:   p.VirtualIP == virtualIP,
		}
	}
	return peers
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
