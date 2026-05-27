// virtualnet/common/common.go
package common

import (
	"encoding/json"
	"time"
)

// 消息类型
const (
	MsgTypeAssignIP    = "assign_ip"    // 分配虚拟IP
	MsgTypeConnect     = "connect"      // 连接指令（服务器下发）
	MsgTypeConnectReq  = "connect_req"  // 连接请求（客户端主动发起）
	MsgTypePeerList    = "peer_list"    // 在线节点列表
	MsgTypeAuthSuccess = "auth_success" // 认证成功
	MsgTypeError       = "error"        // 错误信息
	MsgTypePing        = "ping"         // 服务端下发ping消息
	MsgTypePong        = "pong"         // 客户端回复pong消息
	MsgTypeLatency     = "latency"      // 服务端下发延迟信息
	MsgRegisterName    = "registerName" // 客户端注册信息
)

// WebSocket消息结构
type WSMessage struct {
	Type    string          `json:"type"`    // 消息类型
	Content json.RawMessage `json:"content"` // 消息内容
}

// 分配IP的内容
type AssignIPContent struct {
	VirtualIP  string `json:"virtual_ip"`  // 分配的虚拟IP
	ServerTCP  string `json:"server_tcp"`  // 服务器TCP地址
	ServerWS   string `json:"server_ws"`   // 服务器WS地址（备用）
	ExpireTime int64  `json:"expire_time"` // IP过期时间（时间戳）
	Token      string `json:"token"`       // 认证Token
}

// 连接指令/请求的内容
type ConnectContent struct {
	ConnectFor string `json:"connect_for"` // 目标虚拟IP
	FromIP     string `json:"from_ip"`     // 源虚拟IP（服务器填充）
}

// 在线节点列表内容
type PeerListContent struct {
	Peers []string `json:"peers"` // 所有在线虚拟IP列表
}

// TCP连接认证信息
type TCPAuth struct {
	VirtualIP  string    `json:"virtual_ip"`  // 本地虚拟IP
	ConnectFor string    `json:"connect_for"` // 目标虚拟IP
	Token      string    `json:"token"`       // 认证令牌
	Timestamp  time.Time `json:"timestamp"`   // 时间戳（防重放）
}

// 错误消息内容
type ErrorContent struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误描述
}

type ClientInfoContent struct {
	Name      string `json:"name"`
	VirtualIP string `json:"virtual_ip"`
	Latency   int64  `json:"latency"`
}

type RegisterInfoContent struct {
	Name string `json:"name"`
}
