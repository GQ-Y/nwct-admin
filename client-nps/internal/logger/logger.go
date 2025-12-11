package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	warnLogger  *log.Logger
	debugLogger *log.Logger
	logFile     *os.File
)

// InitLogger 初始化日志系统
func InitLogger() error {
	logDir := "/var/log/nwct"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "system.log")
	
	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
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

// Info 记录信息日志
func Info(format string, v ...interface{}) {
	if infoLogger != nil {
		infoLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Error 记录错误日志
func Error(format string, v ...interface{}) {
	if errorLogger != nil {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Warn 记录警告日志
func Warn(format string, v ...interface{}) {
	if warnLogger != nil {
		warnLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Debug 记录调试日志
func Debug(format string, v ...interface{}) {
	if debugLogger != nil {
		debugLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Fatal 记录致命错误并退出
func Fatal(format string, v ...interface{}) {
	if errorLogger != nil {
		errorLogger.Output(2, fmt.Sprintf(format, v...))
	}
	os.Exit(1)
}

// Close 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
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
		oldPath := filepath.Join("/var/log/nwct", "system.log")
		newPath := filepath.Join("/var/log/nwct", fmt.Sprintf("system.%s.log", time.Now().Format("20060102-150405")))
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
	}

	return nil
}

