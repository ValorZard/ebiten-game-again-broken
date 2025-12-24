package common

type NetPacket struct {
	Username  string
	Tick      uint64
	PositionX float64
	PositionY float64
}

const MAX_NET_PACKET_TICK = 65535
