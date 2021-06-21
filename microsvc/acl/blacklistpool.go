package acl

import (
	"sync"
	"time"
)

// BlacklistPool pool of black ip
type BlacklistPool interface {
	IsBlocked(remoteIP string) bool
	PushBlocked(remoteIP string, expireTs int64)
}

func NewBlacklistPool() BlacklistPool {
	return &blacklistPool{
		blockingIPs: map[string]int64{},
	}
}

type blacklistPool struct {
	blockingIPs map[string]int64 // map[IP]expireTimestampAsSecond
	mu          sync.RWMutex
}

// IsBlocked by remote IP and current timestamp
func (p *blacklistPool) IsBlocked(remoteIP string) bool {
	if nil == p.blockingIPs {
		return false
	}
	p.mu.RLock()
	expires, ok := p.blockingIPs[remoteIP]
	p.mu.RUnlock()
	if ok {
		if 0 == expires {
			return true
		}
		if time.Now().Unix() > expires {
			p.mu.Lock()
			delete(p.blockingIPs, remoteIP)
			p.mu.Unlock()
			return false
		}
		return true
	}
	return false
}

// PushBlocked by remote IP and expires timestamp, if expires timestamp is zero, blocking all the time
func (p *blacklistPool) PushBlocked(remoteIP string, expireTs int64) {
	p.mu.Lock()
	if nil == p.blockingIPs {
		p.blockingIPs = map[string]int64{}
	}
	p.blockingIPs[remoteIP] = expireTs
	p.mu.Unlock()
}
