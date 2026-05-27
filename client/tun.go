package client

import (
	"log"
	"os/exec"
	"sync/atomic"

	"golang.zx2c4.com/wireguard/tun"
)

var (
	readTunCount  atomic.Int64
	writeTunCount atomic.Int64
)

// 初始化Tun设备并配置IP
func initTunDevice(ip string) error {
	// 创建Wintun设备（名称建议固定，避免冲突）
	device, err := tun.CreateTUN("wintun0", 1500) // MTU设置为1500
	if err != nil {
		return err
	}
	tunDevice = device

	// 配置Tun设备IP（Windows使用netsh命令）
	// 虚拟IP属于192.168.32.0/24网段，子网掩码固定为255.255.255.0
	err = setupTunWindows("wintun0", ip, "255.255.255.0")
	if err != nil {
		tunDevice.Close()
		return err
	}

	name, _ := device.Name()
	log.Printf("Tun设备初始化成功: %s, IP: %s", name, ip)
	return nil
}

func addReadTunCount(count int64) {
	readTunCount.Add(count)
}

func addWriteTunCount(count int64) {
	writeTunCount.Add(count)
}

// Windows下配置Tun设备IP和子网掩码
func setupTunWindows(ifname, ip, netmask string) error {
	// 使用netsh命令配置静态IP（需要管理员权限）
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		ifname, "static", ip, netmask)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "配置Tun IP失败, 输出: %s", string(output))
	}
	return nil
}

// GetReadPacketNum 返回开启至今读取包体字节数
func GetReadPacketNum() int64 {
	return readTunCount.Load()
}

// GetSendPacketNum 返回开启至今读取包体字节数
func GetSendPacketNum() int64 {
	return writeTunCount.Load()
}
