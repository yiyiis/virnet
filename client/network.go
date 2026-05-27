package client

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/ipv4"
	"virtualnet/common"
)

// 从Tun设备读取IP包并转发到对应TCP连接
func readTunAndForward() {
	if tunDevice == nil {
		log.Println("Tun设备未初始化，无法读取数据包")
		return
	}

	mtu, _ := tunDevice.MTU()
	packetBuf := make([]byte, mtu) // 存储Tun读取的IP包
	sizes := make([]int, 1)        // 用于tun.Read的长度接收

	for {
		// 从Tun设备读取数据包
		_, err := tunDevice.Read([][]byte{packetBuf}, sizes, 0)
		if err != nil {
			log.Printf("Tun设备读取失败: %v，停止转发", err)
			return
		}
		packetLen := sizes[0]
		if packetLen == 0 {
			continue
		}
		packet := packetBuf[:packetLen]

		// 解析IPv4头部（只处理IPv4包）
		ipHeader, err := ipv4.ParseHeader(packet)
		if err != nil {
			continue
		}

		addReadTunCount(int64(packetLen))

		// 目标IP是虚拟网段内的客户端IP才转发（192.168.32.0/24）
		dstIP := ipHeader.Dst.String()
		var tcpWriter io.Writer
		if !isVirtualIP(dstIP) {
			continue
		}

		// 查找目标IP对应的TCP连接
		mutex.Lock()
		if len(tcpConns) == 0 {
			mutex.Unlock()
			continue
		}

		// 广播地址处理
		if dstIP == "192.168.32.255" && len(tcpConns) > 0 {
			var writers = make([]io.Writer, 0, len(tcpConns))
			for _, client := range tcpConns {
				if client.conn == nil {
					continue
				}
				writers = append(writers, client.conn)
			}
			tcpWriter = io.MultiWriter(writers...)
		} else {
			client, exists := tcpConns[dstIP]
			if exists {
				tcpWriter = client.conn
			}
		}

		mutex.Unlock()

		if tcpWriter == nil {
			log.Printf("未找到与 %s 的TCP连接，无法转发", dstIP)
			continue
		}

		// 发送格式：前4字节（大端）表示IP包长度， followed by IP包数据
		lengthBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lengthBuf, uint32(packetLen))

		// 先发送长度，再发送数据包
		_, err = tcpWriter.Write(append(lengthBuf, packet...))
		if err != nil {
			log.Printf("向 %s 发送数据包失败: %v", dstIP, err)
			// 连接异常，清理该TCP连接
			closeTCPConn(dstIP)
		}
	}
}

// 从TCP连接读取数据并写入Tun设备
func readTCPAndWriteTun(targetIP string, tcpConn net.Conn) {
	defer func() {
		log.Printf("与 %s 的TCP读取协程退出", targetIP)
		closeTCPConn(targetIP) // 退出时清理连接
	}()

	lengthBuf := make([]byte, 4) // 存储前4字节的长度
	for {
		// 先读取4字节长度
		_, err := io.ReadFull(tcpConn, lengthBuf)
		if err != nil {
			log.Printf("从 %s 读取长度失败: %v", targetIP, err)
			return
		}
		packetLen := binary.BigEndian.Uint32(lengthBuf)
		if packetLen == 0 || packetLen > 1500 { // 限制最大长度（MTU=1500）
			log.Printf("无效的数据包长度: %d", packetLen)
			return
		}

		// 读取对应长度的IP包
		packet := make([]byte, packetLen)
		_, err = io.ReadFull(tcpConn, packet)
		if err != nil {
			log.Printf("从 %s 读取数据包失败: %v", targetIP, err)
			return
		}

		addWriteTunCount(int64(packetLen))

		// 写入Tun设备（让系统处理该IP包）
		_, err = tunDevice.Write([][]byte{packet}, 0)
		if err != nil {
			log.Printf("写入Tun设备失败: %v", err)
			return
		}
	}
}

// 建立TCP连接到目标节点（通过服务器中转）
func establishTCPConnection(targetIP string) {
	mutex.Lock()
	localIP := virtualIP
	tcpAddr := serverTCP
	currentToken := token
	mutex.Unlock()

	if localIP == "" || tcpAddr == "" || currentToken == "" {
		log.Println("未初始化完成，无法建立连接")
		return
	}

	// 检查是否已存在连接（避免重复连接）
	mutex.Lock()
	if _, exists := tcpConns[targetIP]; exists {
		mutex.Unlock()
		log.Printf("已与 %s 建立连接，无需重复连接", targetIP)
		return
	}
	mutex.Unlock()

	log.Printf("Tcp 连接服务器: %s\n", tcpAddr)

	// 连接服务器TCP端口（设置5秒超时）
	conn, err := net.DialTimeout("tcp", tcpAddr, 5*time.Second)
	if err != nil {
		log.Printf("连接TCP服务器失败: %v", err)
		return
	}

	// 发送认证信息
	auth := common.TCPAuth{
		VirtualIP:  localIP,
		ConnectFor: targetIP,
		Token:      currentToken,
		Timestamp:  time.Now(),
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(auth); err != nil {
		log.Printf("发送认证信息失败: %v", err)
		conn.Close()
		return
	}

	// 存储连接并启动TCP读取协程（转发到Tun）
	mutex.Lock()
	tcpConns[targetIP] = &Client{
		Name:      strconv.Itoa(len(tcpConns) + 1),
		VirtualIp: targetIP,
		conn:      conn,
	}
	mutex.Unlock()

	log.Printf("已向 %s 发起TCP连接，等待响应...", targetIP)
	// 启动该TCP连接的读取协程（从TCP读数据并写入Tun）
	go readTCPAndWriteTun(targetIP, conn)
}

// 关闭TCP连接并从映射中移除
func closeTCPConn(targetIP string) {
	mutex.Lock()
	defer mutex.Unlock()

	if client, exists := tcpConns[targetIP]; exists {
		client.conn.Close()
		delete(tcpConns, targetIP)
		log.Printf("已关闭与 %s 的TCP连接", targetIP)
	}
}
