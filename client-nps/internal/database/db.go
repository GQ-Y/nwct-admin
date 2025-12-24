package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// InitDB 初始化数据库
func InitDB(dbPath string) (*sql.DB, error) {
	// 允许通过环境变量覆盖
	if v := strings.TrimSpace(os.Getenv("NWCT_DB_PATH")); v != "" {
		dbPath = v
	}

	// 创建数据库目录
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// 如果无权限（如开发机），自动降级到临时目录
		fallback := filepath.Join(os.TempDir(), "nwct", filepath.Base(dbPath))
		fallbackDir := filepath.Dir(fallback)
		if err2 := os.MkdirAll(fallbackDir, 0755); err2 != nil {
			return nil, err
		}
		dbPath = fallback
	}

	// 打开数据库，设置内存优化参数
	database, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=1&_journal_mode=WAL&_cache_size=-2000&_synchronous=NORMAL")
	if err != nil {
		return nil, err
	}

	db = database

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// 设置 SQLite 内存优化参数（减少缓存占用）
	// _cache_size=-2000 表示 2MB 缓存（默认是 -2000KB，即约 2MB）
	// _journal_mode=WAL 使用 WAL 模式，性能更好且内存占用更可控
	// _synchronous=NORMAL 平衡性能和安全性
	if _, err := db.Exec("PRAGMA cache_size = -2000"); err != nil {
		// 忽略错误，继续执行
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		// 忽略错误，继续执行
	}
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		// 忽略错误，继续执行
	}

	// 创建表
	if err := createTables(); err != nil {
		return nil, err
	}

	return db, nil
}

// createTables 创建数据库表
func createTables() error {
	// 设备表
	devicesTable := `
	CREATE TABLE IF NOT EXISTS devices (
		ip TEXT PRIMARY KEY,
		mac TEXT NOT NULL,
		name TEXT,
		vendor TEXT,
		model TEXT,
		type TEXT,
		os TEXT,
		extra TEXT,
		status TEXT DEFAULT 'offline',
		first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// 设备端口表
	devicePortsTable := `
	CREATE TABLE IF NOT EXISTS device_ports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_ip TEXT NOT NULL,
		port INTEGER NOT NULL,
		protocol TEXT NOT NULL,
		service TEXT,
		version TEXT,
		status TEXT DEFAULT 'open',
		scanned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (device_ip) REFERENCES devices(ip) ON DELETE CASCADE,
		UNIQUE(device_ip, port, protocol)
	);`

	// 设备历史表
	deviceHistoryTable := `
	CREATE TABLE IF NOT EXISTS device_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_ip TEXT NOT NULL,
		status TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (device_ip) REFERENCES devices(ip) ON DELETE CASCADE
	);`

	// MQTT日志表
	mqttLogsTable := `
	CREATE TABLE IF NOT EXISTS mqtt_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		direction TEXT NOT NULL,
		topic TEXT NOT NULL,
		qos INTEGER DEFAULT 0,
		payload TEXT,
		status TEXT DEFAULT 'success'
	);`

	// FRP隧道表
	frpTunnelsTable := `
	CREATE TABLE IF NOT EXISTS frp_tunnels (
		name TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		local_ip TEXT NOT NULL,
		local_port INTEGER NOT NULL,
		remote_port INTEGER DEFAULT 0,
		domain TEXT,
		fallback_enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	tables := []string{
		devicesTable,
		devicePortsTable,
		deviceHistoryTable,
		mqttLogsTable,
		frpTunnelsTable,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("创建表失败: %v", err)
		}
	}

	// 轻量迁移：旧库补字段
	if err := ensureColumn("devices", "model", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn("devices", "extra", "TEXT"); err != nil {
		return err
	}
	// 轻量迁移：旧库补字段
	_ = ensureColumn("frp_tunnels", "fallback_enabled", "INTEGER DEFAULT 1")

	return nil
}

func ensureColumn(table, col, colType string) error {
	// 检查是否存在该列
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if strings.EqualFold(name, col) {
			return nil
		}
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, colType))
	if err != nil {
		return fmt.Errorf("迁移失败: %s.%s: %v", table, col, err)
	}
	return nil
}

// GetDB 获取数据库连接
func GetDB() *sql.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
