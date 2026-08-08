//go:build darwin

package client

import (
	"os/exec"
)

func tunDeviceName() string {
	return "utun"
}

// tunPacketOffset 返回 TUN 读写时需预留的协议头空间。
// macOS utun 协议头为 4 字节，wireguard/tun 内部据此做 buf[offset-4:]。
func tunPacketOffset() int {
	return 4
}

func setupTun(ifname, ip, _ string) error {
	// macOS utun 为点对点设备：先挂 /32 地址，再显式添加虚拟网段路由
	cmd := exec.Command("ifconfig", ifname, "inet", ip, ip, "netmask", "255.255.255.255", "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "配置Tun IP失败, 输出: %s", string(output))
	}

	// 先清理可能残留的旧网段路由（旧实例崩溃残留 / 同机多实例冲突）。
	// route delete 在路由不存在时返回非 0，属正常情况，忽略其错误与输出。
	exec.Command("route", "-n", "delete", "-net", "192.168.32.0/24").Run()

	cmd = exec.Command("route", "-n", "add", "-net", "192.168.32.0/24", "-interface", ifname)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return wrapErrorf(err, "添加Tun路由失败, 输出: %s", string(output))
	}
	return nil
}
