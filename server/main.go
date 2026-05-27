package main

import (
	"log"
	"net"
	"net/http"
)

var (
	tcpListener net.Listener                        // TCP监听
	wsAddr      string                              // WebSocket地址
	tcpAddr     string       = "8.138.243.253:8081" // TCP地址
)

func main() {
	initIPPool()

	// 启动TCP服务器
	var err error
	tcpListener, err = net.Listen("tcp", ":8081")
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

	// 启动WebSocket服务器
	wsAddr = ":8080"
	http.HandleFunc("/ws", wsHandler)
	log.Printf("WebSocket服务启动: %s", wsAddr)
	log.Fatal(http.ListenAndServe(wsAddr, nil))
}
