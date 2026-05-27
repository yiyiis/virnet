package client

import (
	"log"
	"sync/atomic"

	"golang.zx2c4.com/wireguard/tun"
)

var (
	readTunCount  atomic.Int64
	writeTunCount atomic.Int64
)

func initTunDevice(ip string) error {
	devName := tunDeviceName()
	device, err := tun.CreateTUN(devName, 1500)
	if err != nil {
		return err
	}
	tunDevice = device

	actualName, _ := device.Name()
	err = setupTun(actualName, ip, "255.255.255.0")
	if err != nil {
		tunDevice.Close()
		return err
	}

	log.Printf("Tun设备初始化成功: %s, IP: %s", actualName, ip)
	return nil
}

func addReadTunCount(count int64) {
	readTunCount.Add(count)
}

func addWriteTunCount(count int64) {
	writeTunCount.Add(count)
}

func GetReadPacketNum() int64 {
	return readTunCount.Load()
}

func GetSendPacketNum() int64 {
	return writeTunCount.Load()
}
