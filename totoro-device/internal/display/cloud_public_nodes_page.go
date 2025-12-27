package display

import (
	"encoding/json"
	"fmt"
	"strings"

	"totoro-device/config"
	"totoro-device/internal/bridgeclient"
)

// CloudPublicNodesPage 公开节点列表（MVP：只展示列表与状态；连接后续加）
type CloudPublicNodesPage struct {
	BasePage
	listView  *ListView
	navBar    *NavBar
	pm        *PageManager
	services  *AppServices
	lastErr   string
}

func NewCloudPublicNodesPage(pm *PageManager) *CloudPublicNodesPage {
	p := &CloudPublicNodesPage{
		BasePage: BasePage{Name: "cloud_public_nodes"},
		pm:       pm,
	}
	p.navBar = NewNavBar("公开节点", true, 480)
	p.navBar.SetOnBack(func() { pm.Back() })
	p.listView = NewListView(0, 60, 480, 420)
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
	p.listView.Clear()

	if p.services == nil || p.services.Config == nil {
		p.lastErr = "服务未初始化"
		p.listView.AddItem(&ListItem{Title: "无法加载", Subtitle: p.lastErr})
		return
	}
	sess, err := p.services.GetBridgeSession()
	if err != nil || sess == nil || strings.TrimSpace(sess.DeviceToken) == "" {
		p.lastErr = "桥梁未注册（缺少 device_token）"
		p.listView.AddItem(&ListItem{Title: "未就绪", Subtitle: p.lastErr})
		return
	}

	bc := &bridgeclient.Client{
		BaseURL:     config.ResolveBridgeBase(p.services.Config),
		DeviceToken: strings.TrimSpace(sess.DeviceToken),
		DeviceID:    strings.TrimSpace(sess.DeviceID),
	}
	nodes, err := bc.GetPublicNodes()
	if err != nil {
		p.lastErr = err.Error()
		p.listView.AddItem(&ListItem{Title: "加载失败", Subtitle: p.lastErr})
		return
	}
	if len(nodes) == 0 {
		p.listView.AddItem(&ListItem{Title: "暂无公开节点", Subtitle: "请稍后刷新"})
		return
	}

	for _, it := range nodes {
		// best-effort：尝试解析常见字段
		b, _ := json.Marshal(it)
		var n struct {
			NodeID  string `json:"node_id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			Region  string `json:"region"`
			ISP     string `json:"isp"`
		}
		_ = json.Unmarshal(b, &n)

		title := strings.TrimSpace(n.Name)
		if title == "" {
			title = strings.TrimSpace(n.NodeID)
		}
		sub := fmt.Sprintf("状态:%s", strings.TrimSpace(n.Status))
		if strings.TrimSpace(n.Region) != "" || strings.TrimSpace(n.ISP) != "" {
			sub = sub + " | " + strings.TrimSpace(n.Region) + " " + strings.TrimSpace(n.ISP)
		}
		p.listView.AddItem(&ListItem{
			Title:     title,
			Subtitle:  strings.TrimSpace(sub),
			ShowArrow: false,
		})
	}
}

func (p *CloudPublicNodesPage) Render(g *Graphics) error {
	g.DrawRect(0, 0, 480, 480, ColorBackgroundStart)
	p.listView.Render(g)
	p.navBar.Render(g)
	return nil
}

func (p *CloudPublicNodesPage) HandleTouch(x, y int, touchType TouchType) bool {
	if p.navBar.HandleTouch(x, y, touchType) {
		return true
	}
	return p.listView.HandleTouch(x, y, touchType)
}


