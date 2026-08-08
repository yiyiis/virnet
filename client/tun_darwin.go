//go:build darwin

package client

import (
	"os/exec"
)

func tunDeviceName() string {
	return "utun"
}

func setupTun(ifname, ip, _ string) error {
	// macOS utun 为点对点设备：先挂 /32 地址，再显式添加虚拟网段路由
	cmd := exec.Command("ifconfig", ifname, "inet", ip, ip, "netmask", "255.255.255.255", "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "配置Tun IP失败, 输出: %s", string(output))
	}

	cmd = exec.Command("route", "-n", "add", "-net", "192.168.32.0/24", "-interface", ifname)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "添加Tun路由失败, 输出: %s", string(output))
	}
	return nil
}
