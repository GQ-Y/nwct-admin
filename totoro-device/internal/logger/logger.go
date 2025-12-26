package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	warnLogger  *log.Logger
	debugLogger *log.Logger
	logFile     *os.File

	logMu           sync.Mutex
	lastRotateCheck int64 // unix nano
)

const (
	// MaxLogSizeBytes 单文件最大 5MB，超过就轮转
	MaxLogSizeBytes int64 = 5 * 1024 * 1024
	// MaxRotatedFiles 保留最近 N 份轮转文件（不含当前 system.log）
	MaxRotatedFiles = 2
)

// InitLogger 初始化日志系统
func InitLogger() error {
	// 优先使用环境变量指定日志目录；否则使用默认 /var/log/nwct；
	// 如果无权限（如 macOS 非root），自动降级到临时目录。
	logDir := os.Getenv("NWCT_LOG_DIR")
	if logDir == "" {
		logDir = "/var/log/nwct"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fallback := filepath.Join(os.TempDir(), "nwct")
		if err2 := os.MkdirAll(fallback, 0755); err2 != nil {
			return err
		}
		logDir = fallback
	}

	logPath := filepath.Join(logDir, "system.log")

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// 目录可能创建成功，但文件打开仍可能因权限失败（macOS 非 root 常见）
		fallback := filepath.Join(os.TempDir(), "nwct")
		_ = os.MkdirAll(fallback, 0755)
		logDir = fallback
		logPath = filepath.Join(logDir, "system.log")
		file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
	}
	logFile = file

	// 创建多写入器（同时写入文件和控制台）
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// 初始化不同级别的日志记录器
	infoLogger = log.New(multiWriter, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
	warnLogger = log.New(multiWriter, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	debugLogger = log.New(multiWriter, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

func maybeRotateLocked() {
	// 限流：最多 1 秒检查一次，避免每条日志都 stat
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&lastRotateCheck)
	if last != 0 && now-last < int64(time.Second) {
		return
	}
	atomic.StoreInt64(&lastRotateCheck, now)
	_ = RotateLog(MaxLogSizeBytes)
}

// Info 记录信息日志
func Info(format string, v ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	maybeRotateLocked()
	if infoLogger != nil {
		infoLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Error 记录错误日志
func Error(format string, v ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	maybeRotateLocked()
	if errorLogger != nil {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Warn 记录警告日志
func Warn(format string, v ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	maybeRotateLocked()
	if warnLogger != nil {
		warnLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Debug 记录调试日志
func Debug(format string, v ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	maybeRotateLocked()
	if debugLogger != nil {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Fatal 记录致命错误并退出
func Fatal(format string, v ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	maybeRotateLocked()
	if errorLogger != nil {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
	os.Exit(1)
}

// Close 关闭日志文件
func Close() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
	}
}

// CurrentLogPath 返回当前实际写入的日志文件路径（优先返回已打开的 logFile）
func CurrentLogPath() string {
	if logFile != nil {
		if name := strings.TrimSpace(logFile.Name()); name != "" {
			return name
		}
	}
	// 兜底：按照 InitLogger 同样的规则推断
	logDir := os.Getenv("NWCT_LOG_DIR")
	if logDir == "" {
		logDir = "/var/log/nwct"
	}
	return filepath.Join(logDir, "system.log")
}

func cleanupRotatedLogs(dir string) {
	// 清理旧轮转日志：system.YYYYMMDD-HHMMSS.log
	matches, _ := filepath.Glob(filepath.Join(dir, "system.*.log"))
	if len(matches) <= MaxRotatedFiles {
		return
	}
	// 按修改时间排序，保留最新 MaxRotatedFiles
	type fi struct {
		path string
		mod  time.Time
	}
	arr := make([]fi, 0, len(matches))
	for _, p := range matches {
		if st, err := os.Stat(p); err == nil {
			arr = append(arr, fi{path: p, mod: st.ModTime()})
		}
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].mod.After(arr[j].mod) })
	for i := MaxRotatedFiles; i < len(arr); i++ {
		_ = os.Remove(arr[i].path)
	}
}

// RotateLog 日志轮转（按大小）
func RotateLog(maxSize int64) error {
	if logFile == nil {
		return nil
	}

	stat, err := logFile.Stat()
	if err != nil {
		return err
	}

	if stat.Size() >= maxSize {
		// 关闭当前日志文件
		logFile.Close()

		// 重命名当前日志文件
		// 以当前 logFile 的目录为准，避免写死 /var/log/nwct
		baseDir := filepath.Dir(logFile.Name())
		oldPath := filepath.Join(baseDir, "system.log")
		newPath := filepath.Join(baseDir, fmt.Sprintf("system.%s.log", time.Now().Format("20060102-150405")))
		os.Rename(oldPath, newPath)

		// 重新打开日志文件
		file, err := os.OpenFile(oldPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		logFile = file

		// 更新日志记录器
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		infoLogger.SetOutput(multiWriter)
		errorLogger.SetOutput(multiWriter)
		warnLogger.SetOutput(multiWriter)
		debugLogger.SetOutput(multiWriter)

		// 清理旧轮转日志，避免本地堆积
		cleanupRotatedLogs(baseDir)
	}

	return nil
}
