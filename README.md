# KvirNet

基于 TUN 虚拟网卡的轻量级虚拟局域网，通过中心化中继服务器实现跨网络设备互联。

## 工作原理

```
┌──────────┐       WebSocket(信令)       ┌──────────┐
│ Client A │ ◄─────────────────────────► │          │
│  (TUN)   │                             │  Server  │
└────┬─────┘       TCP(数据转发)         │          │
     │ ◄──────────────────────────────►  │          │
     │                                   └────┬─────┘
     │                                        │
     │        TCP(数据转发)                    │
     │ ◄───────────────────────────────────►  │
┌────┴─────┐                                 │
│ Client B │                                │
│  (TUN)   │                                │
└──────────┘
```

1. **信令通道**：客户端通过 WebSocket 连接服务器，完成 IP 分配、连接协调、延迟测量
2. **数据通道**：服务器协调客户端建立 TCP 连接，通过 `io.Copy` 双向转发流量
3. **虚拟网卡**：客户端创建 TUN 设备（192.168.32.0/24），拦截 IP 包并转发到 TCP 连接

## 技术栈

- **后端**：Go 1.25
- **虚拟网卡**：WireGuard 的 Wintun 驱动
- **前端**：React + TypeScript + Ant Design（Wails v3 桌面应用）
- **通信协议**：WebSocket（信令）+ TCP（数据）

## 项目结构

```
gamevpn/
├── server/             # 服务端
│   ├── main.go         # 入口，启动 TCP + WebSocket 服务
│   ├── client.go       # 客户端管理、IP 池、连接协调
│   ├── ws_handler.go   # WebSocket 连接处理
│   ├── tcp_handler.go  # TCP 连接认证与数据转发
│   └── message.go      # 消息广播
├── client/             # 客户端核心库
│   ├── client.go       # 客户端结构体、启动逻辑
│   ├── network.go      # TUN 读写、TCP 连接管理、IP 包转发
│   ├── tun.go          # TUN 设备初始化（Windows）
│   ├── message.go      # WebSocket 消息处理
│   └── util.go         # 工具函数
├── common/             # 共用协议定义
│   └── common.go       # 消息类型、数据结构
├── kvirnet/            # Wails 桌面客户端
│   ├── main.go         # Wails 应用入口
│   ├── client_service.go # 前端绑定接口
│   └── frontend/       # React 前端
└── sh/
    └── ci_server.sh    # 服务端部署脚本
```

## 运行

### 服务端

```bash
cd server
go build
./server
# TCP: :8081  WebSocket: :8080
```

### 桌面客户端

```bash
cd kvirnet
wails3 dev
```

### 命令行客户端

```bash
cd client
go run . -server ws://<服务器地址>:8080/ws
```

## 协议

数据通道使用 4 字节大端长度前缀的帧协议：

```
[4 bytes: packet length (big endian)] [IP packet data]
```

信令通道使用 JSON 格式的 WebSocket 消息：

| 类型 | 方向 | 用途 |
|------|------|------|
| assign_ip | Server -> Client | 分配虚拟 IP |
| connect | Server -> Client | 指示建立 TCP 连接 |
| peer_list | Server -> Client | 在线节点列表 |
| ping/pong | 双向 | 延迟测量 |
| latency | Server -> Client | 节点延迟信息 |
| registerName | Client -> Server | 注册昵称 |

## 限制

- 仅支持 Windows（TUN 配置依赖 netsh）
- 流量经服务器中转，非 P2P 直连
- TCP 数据通道无加密
- 客户端断线无自动重连
