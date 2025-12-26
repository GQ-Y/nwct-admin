package kicker

import (
	"strconv"
	"strings"
	"sync"
	"time"

	frpsserver "github.com/fatedier/frp/server"
)

// 维护 invite_id -> active controls，并支持主动踢下线。
// 目标：
// - 邀请码被撤销：节点端立刻断开所有使用该 invite_id 的设备连接
// - ticket 到期：到期自动断开（避免“过期仍然保持连接”）

type entry struct {
	ctl   *frpsserver.Control
	timer *time.Timer
}

var (
	mu      sync.Mutex
	byInvite = map[string]map[*frpsserver.Control]*entry{}
)

func RegisterControl(ctl *frpsserver.Control) {
	if ctl == nil || ctl.LoginMsg() == nil {
		return
	}
	metas := ctl.LoginMsg().Metas
	inviteID := strings.TrimSpace(metas["totoro_invite_id"])
	if inviteID == "" {
		return
	}

	var exp time.Time
	if s := strings.TrimSpace(metas["totoro_ticket_exp_unix"]); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
			exp = time.Unix(n, 0)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	m := byInvite[inviteID]
	if m == nil {
		m = map[*frpsserver.Control]*entry{}
		byInvite[inviteID] = m
	}
	if _, exists := m[ctl]; exists {
		return
	}

	e := &entry{ctl: ctl}
	// ticket 到期后自动踢下线（加 3 秒容差）
	if !exp.IsZero() {
		d := time.Until(exp.Add(3 * time.Second))
		if d < 0 {
			d = 0
		}
		e.timer = time.AfterFunc(d, func() {
			// 到期仅踢当前 control，避免误伤同 invite 的其他新连接
			_ = KickControl(inviteID, ctl)
		})
	}
	m[ctl] = e
}

func UnregisterControl(ctl *frpsserver.Control) {
	if ctl == nil || ctl.LoginMsg() == nil {
		return
	}
	inviteID := strings.TrimSpace(ctl.LoginMsg().Metas["totoro_invite_id"])
	if inviteID == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	m := byInvite[inviteID]
	if m == nil {
		return
	}
	if e := m[ctl]; e != nil {
		if e.timer != nil {
			e.timer.Stop()
		}
		delete(m, ctl)
	}
	if len(m) == 0 {
		delete(byInvite, inviteID)
	}
}

// KickInvite 断开该 invite_id 下所有 active controls，返回断开数量。
func KickInvite(inviteID string) int {
	inviteID = strings.TrimSpace(inviteID)
	if inviteID == "" {
		return 0
	}
	var targets []*frpsserver.Control

	mu.Lock()
	if m := byInvite[inviteID]; m != nil {
		for ctl, e := range m {
			if e != nil && e.timer != nil {
				e.timer.Stop()
			}
			targets = append(targets, ctl)
		}
		delete(byInvite, inviteID)
	}
	mu.Unlock()

	kicked := 0
	for _, ctl := range targets {
		if ctl != nil {
			_ = ctl.Close()
			kicked++
		}
	}
	return kicked
}

// KickControl 仅断开某个 control（用于到期自动踢下线），返回是否成功踢掉。
func KickControl(inviteID string, ctl *frpsserver.Control) bool {
	if ctl == nil {
		return false
	}
	inviteID = strings.TrimSpace(inviteID)
	if inviteID == "" {
		return false
	}

	mu.Lock()
	m := byInvite[inviteID]
	e := (*entry)(nil)
	if m != nil {
		e = m[ctl]
		delete(m, ctl)
		if len(m) == 0 {
			delete(byInvite, inviteID)
		}
	}
	if e != nil && e.timer != nil {
		e.timer.Stop()
	}
	mu.Unlock()

	_ = ctl.Close()
	return true
}


