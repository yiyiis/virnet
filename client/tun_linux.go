//go:build linux

package client

import (
	"fmt"
	"os/exec"
)

func tunDeviceName() string {
	return "kvirnet0"
}

// tunPacketOffset 返回 TUN 读写时需预留的协议头空间。
// Linux 开启 IFF_VNET_HDR 时，wireguard/tun 的 GRO 要求 offset >= virtioNetHdrLen(10)，
// 否则在 tunDevice.Write 处返回 "invalid offset"；取 4(预留)+10 兼容该场景，
// vnetHdr 未开启时多预留的字节也无害。
func tunPacketOffset() int {
	return 14
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
