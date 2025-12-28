package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DeviceInfo 设备信息（设备号和型号）
type DeviceInfo struct {
	DeviceID    string
	DeviceModel string
	UpdatedAt   int64
}

// GetOrInitDeviceInfo 获取或初始化设备信息（从编译时注入的值）
// 如果数据库中没有设备信息，则插入编译时注入的值（deviceID 和 deviceModel）
func GetOrInitDeviceInfo(db *sql.DB, defaultDeviceID, defaultDeviceModel string) (*DeviceInfo, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	// 尝试从数据库读取
	row := db.QueryRow(`SELECT device_id, device_model, updated_at FROM device_info WHERE id=1`)
	var info DeviceInfo
	err := row.Scan(&info.DeviceID, &info.DeviceModel, &info.UpdatedAt)
	if err == nil && strings.TrimSpace(info.DeviceID) != "" && strings.TrimSpace(info.DeviceModel) != "" {
		// 数据库中有值，直接返回
		return &info, nil
	}

	// 数据库中没有值，使用编译时注入的默认值
	deviceID := strings.TrimSpace(defaultDeviceID)
	deviceModel := strings.TrimSpace(defaultDeviceModel)
	if deviceID == "" {
		deviceID = "DEV001" // 兜底
	}
	if deviceModel == "" {
		deviceModel = "Unknown" // 兜底
	}

	now := time.Now().Unix()
	_, err = db.Exec(`
INSERT INTO device_info(id, device_id, device_model, updated_at)
VALUES(1,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  device_id=excluded.device_id,
  device_model=excluded.device_model,
  updated_at=excluded.updated_at
`, deviceID, deviceModel, now)
	if err != nil {
		return nil, fmt.Errorf("初始化设备信息失败: %v", err)
	}

	return &DeviceInfo{
		DeviceID:    deviceID,
		DeviceModel: deviceModel,
		UpdatedAt:   now,
	}, nil
}

// GetDeviceInfo 获取设备信息（不初始化，仅读取）
func GetDeviceInfo(db *sql.DB) (*DeviceInfo, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	row := db.QueryRow(`SELECT device_id, device_model, updated_at FROM device_info WHERE id=1`)
	var info DeviceInfo
	err := row.Scan(&info.DeviceID, &info.DeviceModel, &info.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
