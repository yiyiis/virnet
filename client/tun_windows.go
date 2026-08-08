//go:build windows

package client

import (
	"os/exec"
)

func tunDeviceName() string {
	return "wintun0"
}

// tunPacketOffset 返回 TUN 读写时需预留的协议头空间。
// Windows wintun 无协议头校验，offset 仅决定 buf[offset:] 起始位置，4 兼容现有数据格式。
func tunPacketOffset() int {
	return 4
}

func setupTun(ifname, ip, netmask string) error {
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		ifname, "static", ip, netmask)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "配置Tun IP失败, 输出: %s", string(output))
	}
	return nil
}
