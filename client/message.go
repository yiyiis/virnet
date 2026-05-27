package client

import (
	"encoding/json"
	"log"
	"slices"
	"time"

	"github.com/gorilla/websocket"
	"virtualnet/common"
)

// 处理WebSocket消息
func handleWS() {
	for {
		_, data, err := wsConn.ReadMessage()
		if err != nil {
			log.Printf("WS连接断开: %v", err)
			return
		}
		now := time.Now()
		processWSMessage(data)
		log.Printf("ws处理耗时: %v", time.Since(now))
	}
}

func sendRegisterMsg(name string) error {
	content, err := json.Marshal(common.RegisterInfoContent{Name: name})
	if err != nil {
		return err
	}

	data, err := json.Marshal(common.WSMessage{
		Type:    common.MsgRegisterName,
		Content: content,
	})
	if err != nil {
		return err
	}

	return sendWSMessage(data)
}

func sendWSMessage(data []byte) error {
	mutex.Lock()
	defer mutex.Unlock()

	return wsConn.WriteMessage(websocket.TextMessage, data)
}

// 解析WebSocket消息
func processWSMessage(data []byte) {
	var msg common.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	switch msg.Type {
	case common.MsgTypeAssignIP:
		var content common.AssignIPContent
		if err := json.Unmarshal(msg.Content, &content); err != nil {
			log.Printf("解析IP分配消息失败: %v", err)
			return
		}
		mutex.Lock()
		virtualIP = content.VirtualIP
		serverTCP = content.ServerTCP
		token = content.Token
		tcpConns[virtualIP] = &Client{
			Name:      "本机",
			VirtualIp: virtualIP,
		}
		mutex.Unlock()

		// 初始化Tun设备（使用服务端下发的虚拟IP）
		if err := initTunDevice(virtualIP); err != nil {
			log.Fatalf("Tun设备初始化失败: %v", err)
		}
		// 启动Tun读取转发协程
		go readTunAndForward()

		log.Printf("已分配虚拟IP: %s，开始自动连接在线节点...", virtualIP)

	case common.MsgTypeConnect:
		var content common.ConnectContent
		if err := json.Unmarshal(msg.Content, &content); err != nil {
			log.Printf("解析连接指令失败: %v", err)
			return
		}
		log.Printf("收到连接指令，将连接: %s", content.ConnectFor)
		go establishTCPConnection(content.ConnectFor)

	case common.MsgTypeAuthSuccess:
		var content map[string]string
		if err := json.Unmarshal(msg.Content, &content); err != nil {
			log.Printf("解析认证成功消息失败: %v", err)
			return
		}
		log.Println(content["message"])

		notifyClientChange()
	case common.MsgTypeError:
		var content common.ErrorContent
		if err := json.Unmarshal(msg.Content, &content); err != nil {
			log.Printf("解析错误消息失败: %v", err)
			return
		}
		log.Printf("错误: %s (代码: %d)", content.Message, content.Code)
	case common.MsgTypePing:
		var replyMsg = common.WSMessage{
			Type: common.MsgTypePong,
		}
		marshal, err := json.Marshal(replyMsg)
		if err != nil {
			log.Printf("序列化pong回复信息失败: %v", err)
		}
		err = sendWSMessage(marshal)
		if err != nil {
			log.Printf("回复pong失败: %v", err)
		}
	case common.MsgTypeLatency:
		var clientInfos []common.ClientInfoContent
		if err := json.Unmarshal(msg.Content, &clientInfos); err != nil {
			log.Printf("反序列化延迟信息失败: %v", err)
			return
		}

		curInfoIdx := slices.IndexFunc(clientInfos, func(s common.ClientInfoContent) bool {
			return s.VirtualIP == virtualIP
		})
		if curInfoIdx == -1 {
			log.Printf("延迟信息没有自身延迟")
			return
		}

		curInfo := clientInfos[curInfoIdx]
		mutex.Lock()
		for _, clientInfo := range clientInfos {
			client, ok := tcpConns[clientInfo.VirtualIP]
			if !ok {
				continue
			}

			client.SetId(clientInfo.Name)
			if clientInfo.VirtualIP == virtualIP {
				log.Printf("到服务器延迟: %dms", clientInfo.Latency)
				client.SetLatency(int(clientInfo.Latency))
				continue
			}
			log.Printf("延迟 -> %s : %dms", clientInfo.VirtualIP, curInfo.Latency+clientInfo.Latency)
			client.SetLatency(int(curInfo.Latency + clientInfo.Latency))
		}
		mutex.Unlock()

		notifyClientChange()
	}
}
