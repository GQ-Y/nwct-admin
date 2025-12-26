package limits

import "sync"

// Manager 只做进程内计数（MVP/Beta），避免引入复杂的统计系统。
type Manager struct {
	mu           sync.Mutex
	connsByInvite  map[string]int
	proxiesByInvite map[string]int
	nextPortByInvite map[string]int
}

func New() *Manager {
	return &Manager{
		connsByInvite:  map[string]int{},
		proxiesByInvite: map[string]int{},
		nextPortByInvite: map[string]int{},
	}
}

func (m *Manager) CanOpenConn(inviteID string, maxConns int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.connsByInvite[inviteID]
	if maxConns > 0 && cur >= maxConns {
		return false
	}
	m.connsByInvite[inviteID] = cur + 1
	return true
}

func (m *Manager) CloseConn(inviteID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.connsByInvite[inviteID]
	if cur <= 1 {
		delete(m.connsByInvite, inviteID)
		return
	}
	m.connsByInvite[inviteID] = cur - 1
}

func (m *Manager) CanAddProxy(inviteID string, maxProxies int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.proxiesByInvite[inviteID]
	if maxProxies > 0 && cur >= maxProxies {
		return false
	}
	m.proxiesByInvite[inviteID] = cur + 1
	return true
}

func (m *Manager) RemoveProxy(inviteID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.proxiesByInvite[inviteID]
	if cur <= 1 {
		delete(m.proxiesByInvite, inviteID)
		return
	}
	m.proxiesByInvite[inviteID] = cur - 1
}

// NextPort 生成一个端口候选（不保证不冲突；冲突由 frps 侧最终报错）。
func (m *Manager) NextPort(inviteID string, min int, max int) int {
	if min <= 0 || max <= 0 || min > max {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.nextPortByInvite[inviteID]
	if cur < min || cur > max {
		cur = min
	}
	out := cur
	cur++
	if cur > max {
		cur = min
	}
	m.nextPortByInvite[inviteID] = cur
	return out
}


