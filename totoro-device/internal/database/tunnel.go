package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Tunnel 隧道配置（数据库模型，避免循环导入）
type Tunnel struct {
	Name            string `json:"name"`             // 隧道名称，如 "192.168.1.100_80"
	Type            string `json:"type"`             // tcp, udp, http, https, stcp
	LocalIP         string `json:"local_ip"`         // 本地IP（真实目标）
	LocalPort       int    `json:"local_port"`       // 本地端口（真实目标）
	RemotePort      int    `json:"remote_port"`      // 远程端口（0表示自动分配）
	Domain          string `json:"domain,omitempty"` // HTTP类型使用
	CreatedAt       string `json:"created_at"`
	FallbackEnabled bool   `json:"fallback_enabled"` // 目标不可达时展示默认页
}

// SaveTunnel 保存或更新隧道配置
func SaveTunnel(db *sql.DB, tunnel *Tunnel) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	now := time.Now()

	// 检查隧道是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM frp_tunnels WHERE name = ?)", tunnel.Name).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// 更新隧道
		_, err = db.Exec(`
			UPDATE frp_tunnels 
			SET type = ?, local_ip = ?, local_port = ?, remote_port = ?, domain = ?, fallback_enabled = ?, updated_at = ?
			WHERE name = ?
		`, tunnel.Type, tunnel.LocalIP, tunnel.LocalPort, tunnel.RemotePort, tunnel.Domain, tunnel.FallbackEnabled, now, tunnel.Name)
	} else {
		// 插入新隧道
		createdAt := now
		if tunnel.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, tunnel.CreatedAt); err == nil {
				createdAt = t
			}
		}
		_, err = db.Exec(`
			INSERT INTO frp_tunnels (name, type, local_ip, local_port, remote_port, domain, fallback_enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, tunnel.Name, tunnel.Type, tunnel.LocalIP, tunnel.LocalPort, tunnel.RemotePort, tunnel.Domain, tunnel.FallbackEnabled, createdAt, now)
	}

	return err
}

// DeleteTunnel 删除隧道配置
func DeleteTunnel(db *sql.DB, name string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := db.Exec("DELETE FROM frp_tunnels WHERE name = ?", name)
	return err
}

// RenameTunnel 重命名隧道（更新主键 name）
func RenameTunnel(db *sql.DB, oldName, newName string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("隧道名称不能为空")
	}
	if oldName == newName {
		return nil
	}

	// 检查新名称是否已存在
	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM frp_tunnels WHERE name = ?)", newName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("隧道已存在: %s", newName)
	}

	now := time.Now()
	res, err := db.Exec("UPDATE frp_tunnels SET name = ?, updated_at = ? WHERE name = ?", newName, now, oldName)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("隧道不存在: %s", oldName)
	}
	return nil
}

// GetTunnel 获取单个隧道配置
func GetTunnel(db *sql.DB, name string) (*Tunnel, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var tunnel Tunnel
	var createdAt, updatedAt time.Time
	var domain sql.NullString

	err := db.QueryRow(`
		SELECT name, type, local_ip, local_port, remote_port, domain, fallback_enabled, created_at, updated_at
		FROM frp_tunnels
		WHERE name = ?
	`, name).Scan(
		&tunnel.Name,
		&tunnel.Type,
		&tunnel.LocalIP,
		&tunnel.LocalPort,
		&tunnel.RemotePort,
		&domain,
		&tunnel.FallbackEnabled,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("隧道不存在: %s", name)
		}
		return nil, err
	}

	if domain.Valid {
		tunnel.Domain = domain.String
	}
	tunnel.CreatedAt = createdAt.Format(time.RFC3339)

	return &tunnel, nil
}

// GetAllTunnels 获取所有隧道配置
func GetAllTunnels(db *sql.DB) ([]*Tunnel, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	rows, err := db.Query(`
		SELECT name, type, local_ip, local_port, remote_port, domain, fallback_enabled, created_at, updated_at
		FROM frp_tunnels
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tunnels []*Tunnel
	for rows.Next() {
		var tunnel Tunnel
		var createdAt, updatedAt time.Time
		var domain sql.NullString

		err := rows.Scan(
			&tunnel.Name,
			&tunnel.Type,
			&tunnel.LocalIP,
			&tunnel.LocalPort,
			&tunnel.RemotePort,
			&domain,
			&tunnel.FallbackEnabled,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			continue
		}

		if domain.Valid {
			tunnel.Domain = domain.String
		}
		tunnel.CreatedAt = createdAt.Format(time.RFC3339)

		tunnels = append(tunnels, &tunnel)
	}

	return tunnels, rows.Err()
}
