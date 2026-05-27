package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"virtualnet/common"
)

// 发送WebSocket消息
func sendWSMessage(client *Client, msgType string, content []byte) error {
	wsMux.Lock()
	defer wsMux.Unlock()

	msg := common.WSMessage{
		Type:    msgType,
		Content: content,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return client.wsConn.WriteMessage(websocket.TextMessage, data)
}

func internalBroadcastClientInfo() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		broadcastClientInfo()

		<-ticker.C
	}
}

// 广播信息给所有客户端
func broadcastClientInfo() {

	var clientArr []*Client
	mutex.Lock()
	for _, client := range clients {
		clientArr = append(clientArr, client)
	}
	mutex.Unlock()

	var latencies []common.ClientInfoContent
	for _, client := range clientArr {
		client.mu.Lock()
		latencies = append(latencies, common.ClientInfoContent{
			Name:      client.name,
			VirtualIP: client.virtualIP,
			Latency:   client.latency,
		})
		client.mu.Unlock()
	}

	latenciesData, err := json.Marshal(latencies)
	if err != nil {
		log.Printf("序列化延迟信息失败: %v", err)
		return
	}

	for _, client := range clientArr {
		_ = sendWSMessage(client, common.MsgTypeLatency, latenciesData)
	}

}

// 生成错误消息内容
func errorContent(code int, msg string) []byte {
	content, _ := json.Marshal(common.ErrorContent{Code: code, Message: msg})
	return content
}

// 生成错误消息
func errorMessage(code int, msg string) []byte {
	content := errorContent(code, msg)
	msgObj := common.WSMessage{Type: common.MsgTypeError, Content: content}
	data, _ := json.Marshal(msgObj)
	return data
}
