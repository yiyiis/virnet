package main

import (
	"log"
	"net"
	"net/http"

	"virtualnet/common"
)

var (
	tcpListener net.Listener // TCP监听
	wsAddr      string       // WebSocket地址
	tcpAddr     string       // TCP地址（对外宣告的公网IP:端口）
)

func main() {
	// 加载 .env（存在则读取，不存在静默忽略），之后所有配置走环境变量
	common.LoadEnv()

	initIPPool()

	// 服务器地址可通过环境变量配置，未设置时使用默认值
	// VIRNET_TCP_ADDR / VIRNET_WS_ADDR 用于对外宣告的地址（下发给客户端），
	// 实际监听端口从对应地址中解析（支持 "host:port" 或 ":port" 两种写法）。
	tcpAddr = common.Getenv("VIRNET_TCP_ADDR", "43.138.247.132:8081")
	wsAddr = common.Getenv("VIRNET_WS_ADDR", ":8080")

	// 启动TCP服务器（监听端口从 tcpAddr 解析，保持对外宣告地址与实际监听端口一致）
	var err error
	tcpListener, err = net.Listen("tcp", listenAddr(tcpAddr, "8081"))
	if err != nil {
		log.Fatalf("TCP服务启动失败: %v", err)
	}
	log.Printf("TCP服务启动: %s", tcpAddr)

	// 启动TCP连接处理
	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				log.Printf("TCP接受连接失败: %v", err)
				continue
			}
			go handleTCPConnection(conn)
		}
	}()

	go internalBroadcastClientInfo()

	// 启动WebSocket服务器（监听地址同样从 wsAddr 解析端口）
	http.HandleFunc("/ws", wsHandler)
	log.Printf("WebSocket服务启动: %s", wsAddr)
	log.Fatal(http.ListenAndServe(listenAddr(wsAddr, "8080"), nil))
}

// listenAddr 从对外宣告地址（"host:port" 或 ":port"）中取出端口，
// 拼成本机监听地址 ":port"。便于对外宣告公网IP、本机仍监听所有接口。
func listenAddr(announcedAddr, defaultPort string) string {
	_, port, err := net.SplitHostPort(announcedAddr)
	if err != nil || port == "" {
		return ":" + defaultPort
	}
	return ":" + port
}
