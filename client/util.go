package client

import (
	"fmt"
	"net"
)

// 检查IP是否为虚拟网段（192.168.32.0/24）
func isVirtualIP(ip string) bool {
	ipAddr := net.ParseIP(ip).To4()
	if ipAddr == nil {
		return false
	}
	// 192.168.32.0/24的网络地址：前三个字节为192.168.32
	return ipAddr[0] == 192 && ipAddr[1] == 168 && ipAddr[2] == 32
}

// 错误包装工具函数
func wrapErrorf(err error, format string, v ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, v...), err)
}
