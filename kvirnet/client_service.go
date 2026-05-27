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
