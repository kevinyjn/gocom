package pinger

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/kevinyjn/gocom/logger"
)

// Constants
const (
	PingTimeout = 1
)

// ICMP struct
type ICMP struct {
	Type        uint8
	Code        uint8
	Checksum    uint16
	Identifier  uint16
	SequenceNum uint16
}

// Ping IP if pingable
func Ping(ip string) bool {
	// fill data package
	icmp := ICMP{}
	icmp.Type = 8 //8->echo message  0->reply message
	icmp.Code = 0
	icmp.Checksum = 0
	icmp.Identifier = 0
	icmp.SequenceNum = 0

	recvBuf := make([]byte, 32)
	var buffer bytes.Buffer

	// 先在buffer中写入icmp数据报求去校验和
	binary.Write(&buffer, binary.BigEndian, icmp)
	icmp.Checksum = CheckSum(buffer.Bytes())
	// 然后清空buffer并把求完校验和的icmp数据报写入其中准备发送
	buffer.Reset()
	binary.Write(&buffer, binary.BigEndian, icmp)

	conn, err := net.DialTimeout("ip4:icmp", ip, PingTimeout*time.Second)
	if nil != err {
		logger.Warning.Printf("ping %s failed with error:%v", ip, err)
		return false
	}
	_, err = conn.Write(buffer.Bytes())
	if nil != err {
		logger.Warning.Printf("ping %s while write connection icmp buffer failed with error:%v", ip, err)
		return false
	}
	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	num, err := conn.Read(recvBuf)
	if nil != err {
		logger.Warning.Printf("ping %s while read connection deadline failed with error:%v", ip, err)
		return false
	}

	conn.SetReadDeadline(time.Time{})

	if string(recvBuf[0:num]) != "" {
		return true
	}
	return false

}

// CheckSum ping data
func CheckSum(data []byte) uint16 {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += (sum >> 16)

	return uint16(^sum)
}

// Connectable by IP and port
func Connectable(ip string, port int) bool {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, PingTimeout*time.Second)
	if nil != err {
		logger.Warning.Printf("try connect %s failed with error:%v", addr, err)
		return false
	}
	conn.Close()
	return true
}
