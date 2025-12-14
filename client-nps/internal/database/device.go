package database

import (
	"database/sql"
	"fmt"
	"time"
)

type DeviceActivity struct {
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
	Status    string    `json:"status"`
	Name      string    `json:"name"`
	Vendor    string    `json:"vendor"`
	Model     string    `json:"model"`
}

// SaveDevice 保存或更新设备
func SaveDevice(db *sql.DB, device *Device) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	now := time.Now()
	
	// 检查设备是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM devices WHERE ip = ?)", device.IP).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// 更新设备
		_, err = db.Exec(`
			UPDATE devices 
			SET mac = ?, name = ?, vendor = ?, model = ?, type = ?, os = ?, extra = ?, status = ?, last_seen = ?, updated_at = ?
			WHERE ip = ?
		`, device.MAC, device.Name, device.Vendor, device.Model, device.Type, device.OS, device.Extra, device.Status, now, now, device.IP)
	} else {
		// 插入新设备
		_, err = db.Exec(`
			INSERT INTO devices (ip, mac, name, vendor, model, type, os, extra, status, first_seen, last_seen, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, device.IP, device.MAC, device.Name, device.Vendor, device.Model, device.Type, device.OS, device.Extra, device.Status, now, now, now)
	}

	if err != nil {
		return err
	}

	// 记录历史
	_, err = db.Exec(`
		INSERT INTO device_history (device_ip, status, timestamp)
		VALUES (?, ?, ?)
	`, device.IP, device.Status, now)

	return err
}

// UpdateDeviceStatus 更新设备在线状态，并写入历史
func UpdateDeviceStatus(db *sql.DB, ip string, status string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	now := time.Now()
	_, err := db.Exec(`
		UPDATE devices
		SET status = ?, last_seen = ?, updated_at = ?
		WHERE ip = ?
	`, status, now, now, ip)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO device_history (device_ip, status, timestamp)
		VALUES (?, ?, ?)
	`, ip, status, now)
	return err
}

// TouchDeviceLastSeen 仅刷新 last_seen（用于设备仍在线时的心跳刷新）
func TouchDeviceLastSeen(db *sql.DB, ip string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	now := time.Now()
	_, err := db.Exec(`
		UPDATE devices
		SET last_seen = ?, updated_at = ?
		WHERE ip = ?
	`, now, now, ip)
	return err
}

// GetDevice 获取设备
func GetDevice(db *sql.DB, ip string) (*Device, error) {
	device := &Device{}
	err := db.QueryRow(`
		SELECT ip, mac, name, vendor, model, type, os, extra, status, first_seen, last_seen
		FROM devices
		WHERE ip = ?
	`, ip).Scan(
		&device.IP, &device.MAC, &device.Name, &device.Vendor, &device.Model,
		&device.Type, &device.OS, &device.Extra, &device.Status, &device.FirstSeen, &device.LastSeen,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return device, nil
}

// GetDevices 获取设备列表
func GetDevices(db *sql.DB, status, deviceType string, limit, offset int) ([]Device, int, error) {
	query := "SELECT ip, mac, name, vendor, model, type, os, status, first_seen, last_seen FROM devices WHERE 1=1"
	args := []interface{}{}
	// 过滤无意义的广播/占位 MAC（避免 UI 出现 192.168.x.255 / FF:FF:FF:FF:FF:FF 等记录）
	query += " AND mac != ?"
	args = append(args, "FF:FF:FF:FF:FF:FF")

	if status != "" && status != "all" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if deviceType != "" {
		query += " AND type = ?"
		args = append(args, deviceType)
	}

	query += " ORDER BY last_seen DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	devices := []Device{}
	for rows.Next() {
		var device Device
		err := rows.Scan(
			&device.IP, &device.MAC, &device.Name, &device.Vendor, &device.Model,
			&device.Type, &device.OS, &device.Status, &device.FirstSeen, &device.LastSeen,
		)
		if err != nil {
			continue
		}
		devices = append(devices, device)
	}

	// 获取总数
	countQuery := "SELECT COUNT(*) FROM devices WHERE 1=1"
	countArgs := []interface{}{}
	countQuery += " AND mac != ?"
	countArgs = append(countArgs, "FF:FF:FF:FF:FF:FF")
	if status != "" && status != "all" {
		countQuery += " AND status = ?"
		countArgs = append(countArgs, status)
	}
	if deviceType != "" {
		countQuery += " AND type = ?"
		countArgs = append(countArgs, deviceType)
	}

	var total int
	err = db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// GetRecentActivity 返回最近的设备上线/离线历史
func GetRecentActivity(db *sql.DB, limit int) ([]DeviceActivity, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := db.Query(`
		SELECT h.timestamp, h.status, d.ip, d.name, d.vendor, d.model
		FROM device_history h
		JOIN devices d ON d.ip = h.device_ip
		ORDER BY h.timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeviceActivity{}
	for rows.Next() {
		var a DeviceActivity
		if err := rows.Scan(&a.Timestamp, &a.Status, &a.IP, &a.Name, &a.Vendor, &a.Model); err != nil {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

// ClearAllDeviceData 清空本次扫描前的历史设备数据（设备、端口、历史）
// 需求：每次扫描都丢弃历史数据，避免 UI 混入旧网段/旧结果
func ClearAllDeviceData(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	// 顺序：先删子表，再删主表
	if _, err := db.Exec(`DELETE FROM device_ports`); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM device_history`); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM devices`); err != nil {
		return err
	}
	return nil
}

// SaveDevicePort 保存设备端口
func SaveDevicePort(db *sql.DB, deviceIP string, port *DevicePort) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO device_ports (device_ip, port, protocol, service, version, status, scanned_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, deviceIP, port.Port, port.Protocol, port.Service, port.Version, port.Status, time.Now())
	return err
}

// GetDevicePorts 获取设备端口列表
func GetDevicePorts(db *sql.DB, deviceIP string) ([]DevicePort, error) {
	rows, err := db.Query(`
		SELECT id, device_ip, port, protocol, service, version, status, scanned_at
		FROM device_ports
		WHERE device_ip = ?
		ORDER BY port
	`, deviceIP)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := []DevicePort{}
	for rows.Next() {
		var port DevicePort
		err := rows.Scan(
			&port.ID, &port.DeviceIP, &port.Port, &port.Protocol,
			&port.Service, &port.Version, &port.Status, &port.ScannedAt,
		)
		if err != nil {
			continue
		}
		ports = append(ports, port)
	}

	return ports, nil
}

// GetMQTTLogs 获取MQTT日志
func GetMQTTLogs(db *sql.DB, topic, direction string, startTime, endTime time.Time, limit, offset int) ([]MQTTLog, int, error) {
	query := "SELECT id, timestamp, direction, topic, qos, payload, status FROM mqtt_logs WHERE 1=1"
	args := []interface{}{}

	if topic != "" {
		query += " AND topic LIKE ?"
		args = append(args, "%"+topic+"%")
	}

	if direction != "" && direction != "all" {
		query += " AND direction = ?"
		args = append(args, direction)
	}

	if !startTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, startTime)
	}

	if !endTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, endTime)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []MQTTLog{}
	for rows.Next() {
		var log MQTTLog
		err := rows.Scan(
			&log.ID, &log.Timestamp, &log.Direction, &log.Topic,
			&log.QoS, &log.Payload, &log.Status,
		)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	// 获取总数
	countQuery := "SELECT COUNT(*) FROM mqtt_logs WHERE 1=1"
	countArgs := []interface{}{}
	if topic != "" {
		countQuery += " AND topic LIKE ?"
		countArgs = append(countArgs, "%"+topic+"%")
	}
	if direction != "" && direction != "all" {
		countQuery += " AND direction = ?"
		countArgs = append(countArgs, direction)
	}
	if !startTime.IsZero() {
		countQuery += " AND timestamp >= ?"
		countArgs = append(countArgs, startTime)
	}
	if !endTime.IsZero() {
		countQuery += " AND timestamp <= ?"
		countArgs = append(countArgs, endTime)
	}

	var total int
	err = db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

