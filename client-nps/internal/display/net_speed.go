package display

import (
	"fmt"
	"sync"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
)

// NetSpeedSampler 采样网络收发字节并计算速率
type NetSpeedSampler struct {
	mu       sync.Mutex
	lastTime time.Time
	lastRecv uint64
	lastSent uint64
	lastIface string
	inited   bool
}

func NewNetSpeedSampler() *NetSpeedSampler {
	return &NetSpeedSampler{}
}

// SampleKBps 返回 (uploadKBps, downloadKBps)
func (s *NetSpeedSampler) SampleKBps(iface string) (float64, float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return 0, 0, err
	}

	var recv, sent uint64
	found := false
	for _, c := range counters {
		if c.Name == iface {
			recv = c.BytesRecv
			sent = c.BytesSent
			found = true
			break
		}
	}
	if !found {
		// 如果找不到指定网卡，尝试汇总所有
		all, err2 := psnet.IOCounters(false)
		if err2 != nil || len(all) == 0 {
			return 0, 0, fmt.Errorf("找不到网卡计数器: %s", iface)
		}
		recv = all[0].BytesRecv
		sent = all[0].BytesSent
	}

	// 首次采样或网卡发生变化：只记录，不计算
	if !s.inited || s.lastIface != iface || s.lastTime.IsZero() {
		s.inited = true
		s.lastIface = iface
		s.lastTime = now
		s.lastRecv = recv
		s.lastSent = sent
		return 0, 0, nil
	}

	dt := now.Sub(s.lastTime).Seconds()
	if dt <= 0.2 {
		// 时间间隔太短则不更新，避免跳动
		return 0, 0, nil
	}

	dRecv := float64(recv - s.lastRecv)
	dSent := float64(sent - s.lastSent)
	downKBps := dRecv / 1024.0 / dt
	upKBps := dSent / 1024.0 / dt

	s.lastTime = now
	s.lastRecv = recv
	s.lastSent = sent

	return upKBps, downKBps, nil
}


