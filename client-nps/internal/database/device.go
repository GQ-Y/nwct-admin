package database

import (
	"database/sql"
	"time"
)

// SaveDevice 保存或更新设备
func SaveDevice(db *sql.DB, device *Device) error {
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
			SET mac = ?, name = ?, vendor = ?, type = ?, os = ?, status = ?, last_seen = ?, updated_at = ?
			WHERE ip = ?
		`, device.MAC, device.Name, device.Vendor, device.Type, device.OS, device.Status, now, now, device.IP)
	} else {
		// 插入新设备
		_, err = db.Exec(`
			INSERT INTO devices (ip, mac, name, vendor, type, os, status, first_seen, last_seen, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, device.IP, device.MAC, device.Name, device.Vendor, device.Type, device.OS, device.Status, now, now, now)
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

// GetDevice 获取设备
func GetDevice(db *sql.DB, ip string) (*Device, error) {
	device := &Device{}
	err := db.QueryRow(`
		SELECT ip, mac, name, vendor, type, os, status, first_seen, last_seen
		FROM devices
		WHERE ip = ?
	`, ip).Scan(
		&device.IP, &device.MAC, &device.Name, &device.Vendor,
		&device.Type, &device.OS, &device.Status, &device.FirstSeen, &device.LastSeen,
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
	query := "SELECT ip, mac, name, vendor, type, os, status, first_seen, last_seen FROM devices WHERE 1=1"
	args := []interface{}{}

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
			&device.IP, &device.MAC, &device.Name, &device.Vendor,
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

