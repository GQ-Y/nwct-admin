package display

import (
	"encoding/json"
	"fmt"
	"strings"

	"totoro-device/internal/bridgeclient"
)

// CloudPublicNodesPage 公开节点列表（MVP：只展示列表与状态；连接后续加）
type CloudPublicNodesPage struct {
	BasePage
	navBar    *NavBar
	pm        *PageManager
	services  *AppServices
	lastErr   string

	nodes []cloudNodeItem

	// 列表滚动
	offsetY     int
	pressedIdx  int
	dragging    bool
	dragStartY  int
	lastDragY   int

	confirm ConfirmDialog
}

type cloudNodeItem struct {
	NodeID   string
	Name     string
	Status   string
	Region   string
	ISP      string
	Protocols string
}

func NewCloudPublicNodesPage(pm *PageManager) *CloudPublicNodesPage {
	p := &CloudPublicNodesPage{
		BasePage: BasePage{Name: "cloud_public_nodes"},
		pm:       pm,
	}
	p.navBar = NewNavBar("公开节点", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	p.pressedIdx = -1
	return p
}

func (p *CloudPublicNodesPage) SetServices(s *AppServices) {
	p.services = s
	p.refresh()
}

func (p *CloudPublicNodesPage) OnEnter() {
	p.refresh()
}

func (p *CloudPublicNodesPage) refresh() {
	p.lastErr = ""
	p.nodes = nil
	p.offsetY = 0
	p.pressedIdx = -1
	p.dragging = false

	if p.services == nil || p.services.Config == nil {
		p.lastErr = "服务未初始化"
		return
	}

	var rawNodes []any
	err := p.services.RegisterBridgeAndRetryOn401(func(bc *bridgeclient.Client) error {
		ns, e := bc.GetPublicNodes()
		if e != nil {
			return e
		}
		rawNodes = ns
		return nil
	})
	if err != nil {
		p.lastErr = err.Error()
		return
	}
	if len(rawNodes) == 0 {
		return
	}

	for _, it := range rawNodes {
		// best-effort：尝试解析常见字段
		b, _ := json.Marshal(it)
		var n struct {
			NodeID  string `json:"node_id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			Region  string `json:"region"`
			ISP     string `json:"isp"`
			HTTPEnabled  bool `json:"http_enabled"`
			HTTPSEnabled bool `json:"https_enabled"`
			TCPPortPool  *struct{ Min, Max int } `json:"tcp_port_pool"`
			UDPPortPool  *struct{ Min, Max int } `json:"udp_port_pool"`
		}
		_ = json.Unmarshal(b, &n)

		protos := []string{"TCP", "UDP"}
		if n.HTTPEnabled {
			protos = append(protos, "HTTP")
		}
		if n.HTTPSEnabled {
			protos = append(protos, "HTTPS")
		}
		// 若明确没有 UDP pool，可不显示 UDP（更贴近“支持协议”）
		if n.UDPPortPool == nil {
			// 留 TCP
			protos = filterOut(protos, "UDP")
		}
		if n.TCPPortPool == nil {
			protos = filterOut(protos, "TCP")
		}
		if len(protos) == 0 {
			protos = []string{"TCP"}
		}

		p.nodes = append(p.nodes, cloudNodeItem{
			NodeID:   strings.TrimSpace(n.NodeID),
			Name:     strings.TrimSpace(n.Name),
			Status:   strings.TrimSpace(n.Status),
			Region:   strings.TrimSpace(n.Region),
			ISP:      strings.TrimSpace(n.ISP),
			Protocols: strings.Join(protos, "/"),
		})
	}
}

func (p *CloudPublicNodesPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.navBar.Render(g)

	top := 60
	if p.lastErr != "" {
		_ = g.DrawTextTTF("加载失败："+p.lastErr, 24, 86, ColorErrorRed, 14, FontWeightRegular)
	}
	if len(p.nodes) == 0 && p.lastErr == "" {
		_ = g.DrawTextTTF("暂无公开节点", 24, 92, ColorTextSecondary, 14, FontWeightRegular)
	}

	// 列表渲染（简化滚动）
	itemH := 82
	startY := top + p.offsetY
	for i, n := range p.nodes {
		y := startY + i*itemH
		if y+itemH < top || y > 480 {
			continue
		}
		if i == p.pressedIdx {
			g.DrawRect(0, y, 480, itemH, ColorPressed)
		}
		title := n.Name
		if strings.TrimSpace(title) == "" {
			title = n.NodeID
		}
		sub := fmt.Sprintf("状态:%s | %s %s", n.Status, n.Region, n.ISP)
		sub = strings.TrimSpace(strings.ReplaceAll(sub, "  ", " "))
		g.DrawTextTTF(title, 24, y+26, ColorTextPrimary, 18, FontWeightMedium)
		g.DrawTextTTF("协议: "+n.Protocols, 24, y+52, ColorTextSecondary, 14, FontWeightRegular)

		// 右侧连接按钮
		btnW, btnH := 92, 36
		btnX := 480 - 24 - btnW
		btnY := y + (itemH-btnH)/2
		bg := ColorBrandBlue
		if n.Status == "offline" {
			bg = ColorSeparator
		}
		g.DrawRectRounded(btnX, btnY, btnW, btnH, 12, bg)
		lbl := "连接"
		if n.Status == "offline" {
			lbl = "离线"
		}
		lw := g.MeasureText(lbl, 14, FontWeightMedium)
		g.DrawTextTTF(lbl, btnX+(btnW-lw)/2, btnY+(btnH-int(14))/2, ColorBackgroundStart, 14, FontWeightMedium)

		// 分隔线
		if i < len(p.nodes)-1 {
			g.DrawRect(24, y+itemH-1, 480-24, 1, ColorSeparator)
		}
	}

	p.confirm.Render(g)
	return nil
}

func (p *CloudPublicNodesPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.confirm.Visible {
		return p.confirm.HandleTouch(x, y, touchType)
	}
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	top := 60
	if x < 0 || x > 480 || y < top || y > 480 {
		return false
	}

	itemH := 82
	// scroll drag
	if touchType == TouchDown {
		p.dragging = false
		p.dragStartY = y
		p.lastDragY = y
		// 预先标记 pressed
		idx := (y-(top+p.offsetY)) / itemH
		if idx >= 0 && idx < len(p.nodes) {
			p.pressedIdx = idx
		} else {
			p.pressedIdx = -1
		}
		return true
	}
	if touchType == TouchMove {
		dy := y - p.lastDragY
		if !p.dragging {
			d := y - p.dragStartY
			if d < 0 {
				d = -d
			}
			if d > 6 {
				p.dragging = true
			}
		}
		if p.dragging {
			p.offsetY += dy
			maxOffset := 0
			minOffset := -(len(p.nodes)*itemH - (480-top))
			if minOffset > 0 {
				minOffset = 0
			}
			if p.offsetY > maxOffset {
				p.offsetY = maxOffset
			}
			if p.offsetY < minOffset {
				p.offsetY = minOffset
			}
			p.pressedIdx = -1
			p.lastDragY = y
			return true
		}
		p.lastDragY = y
		return true
	}
	if touchType == TouchUp {
		if p.dragging {
			p.dragging = false
			p.pressedIdx = -1
			return true
		}
		idx := (y-(top+p.offsetY)) / itemH
		if idx < 0 || idx >= len(p.nodes) {
			p.pressedIdx = -1
			return true
		}
		if p.pressedIdx != idx {
			p.pressedIdx = -1
			return true
		}
		p.pressedIdx = -1

		n := p.nodes[idx]
		// connect button area
		btnW, btnH := 92, 36
		btnX := 480 - 24 - btnW
		btnY := (top+p.offsetY) + idx*itemH + (itemH-btnH)/2
		if x >= btnX && x <= btnX+btnW && y >= btnY && y <= btnY+btnH {
			if n.Status == "offline" {
				return true
			}
			name := n.Name
			if strings.TrimSpace(name) == "" {
				name = n.NodeID
			}
			p.confirm = ConfirmDialog{
				Visible: true,
				Title:   "连接公开节点",
				Message: "确定连接到「" + name + "」？",
				ConfirmText: "连接",
				CancelText:  "取消",
				OnConfirm: func() {
					if p.services != nil {
						_ = p.services.ConnectPublicNode(n.NodeID)
					}
					_ = p.pm.NavigateTo("cloud_status")
				},
			}
			return true
		}
		return true
	}
	return false
}

func filterOut(in []string, s string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}


