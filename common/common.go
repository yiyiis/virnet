// virtualnet/common/common.go
package common

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// LoadEnv 从工作目录或指定路径加载 .env 文件到当前进程的环境变量。
// 文件不存在时静默忽略（线上部署通常用真实环境变量，无需 .env）。
// 解析失败仅告警不中断，避免本地 .env 笔误导致程序无法启动。
//
// 使用约定：
//   - 本地开发：仓库根放 .env（已在 .gitignore 排除），团队共享 .env.example
//   - 线上部署：不用 .env，用 systemd EnvironmentFile / docker env_file / export
func LoadEnv(paths ...string) {
	if err := godotenv.Load(paths...); err != nil {
		// .env 不存在是正常情况，仅在解析出错时告警
		if !os.IsNotExist(err) {
			log.Printf("加载.env文件失败: %v", err)
		}
	}
}

// Getenv 读取环境变量，未设置时返回 fallback 兜底值。
// 服务端与客户端的所有可配置项（服务器IP/端口等）统一走此函数，
// 既支持用环境变量配置，又保留原有硬编码作为默认值。
func Getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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
