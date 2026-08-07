package main

import "virtualnet/client"

type ClientService struct{}

func (s *ClientService) GetClient() []*client.Client {
	return client.GetClients()
}

func (s *ClientService) SetClientInfo(name string) error {
	return client.SetName(name)
}

func (s *ClientService) GetReadPacketNum() int64 {
	return client.GetReadPacketNum()
}

func (s *ClientService) GetSendPacketNum() int64 {
	return client.GetSendPacketNum()
}

// GetOnlinePeers 获取全部在线节点（含连接状态），供前端展示
func (s *ClientService) GetOnlinePeers() []client.PeerInfo {
	return client.GetOnlinePeers()
}

// ConnectTo 主动连接目标虚拟IP
func (s *ClientService) ConnectTo(ip string) error {
	return client.ConnectTo(ip)
}
