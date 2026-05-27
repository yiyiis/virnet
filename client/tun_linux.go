//go:build linux

package client

import (
	"fmt"
	"os/exec"
)

func tunDeviceName() string {
	return "kvirnet0"
}

func setupTun(ifname, ip, netmask string) error {
	cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/24", ip), "dev", ifname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "配置Tun IP失败, 输出: %s", string(output))
	}

	cmd = exec.Command("ip", "link", "set", ifname, "up")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "启动Tun设备失败, 输出: %s", string(output))
	}
	return nil
}
