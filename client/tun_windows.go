//go:build windows

package client

import (
	"os/exec"
)

func tunDeviceName() string {
	return "wintun0"
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
