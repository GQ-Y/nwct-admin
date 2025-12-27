package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Backlight 基于 sysfs 的背光控制器（Buildroot 常见）
type Backlight struct {
	BaseDir        string // /sys/class/backlight/<name>
	BrightnessPath string
	MaxPath        string
}

func DiscoverBacklight() (*Backlight, error) {
	ents, err := filepath.Glob("/sys/class/backlight/*")
	if err != nil || len(ents) == 0 {
		return nil, fmt.Errorf("未检测到背光设备（/sys/class/backlight）")
	}
	// 取第一个可用的
	for _, d := range ents {
		bp := filepath.Join(d, "brightness")
		mp := filepath.Join(d, "max_brightness")
		if _, err := os.Stat(bp); err != nil {
			continue
		}
		if _, err := os.Stat(mp); err != nil {
			continue
		}
		return &Backlight{BaseDir: d, BrightnessPath: bp, MaxPath: mp}, nil
	}
	return nil, fmt.Errorf("未找到可用背光节点（brightness/max_brightness）")
}

func (b *Backlight) Max() (int, error) {
	v, err := readInt(b.MaxPath)
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, fmt.Errorf("max_brightness 无效: %d", v)
	}
	return v, nil
}

func (b *Backlight) CurrentRaw() (int, error) {
	return readInt(b.BrightnessPath)
}

// SetPercent 设置亮度百分比（0~100）
func (b *Backlight) SetPercent(percent int) error {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	maxV, err := b.Max()
	if err != nil {
		return err
	}
	raw := percent * maxV / 100
	return writeInt(b.BrightnessPath, raw)
}

func (b *Backlight) GetPercent() (int, error) {
	maxV, err := b.Max()
	if err != nil {
		return 0, err
	}
	raw, err := b.CurrentRaw()
	if err != nil {
		return 0, err
	}
	if raw < 0 {
		raw = 0
	}
	if raw > maxV {
		raw = maxV
	}
	return int(float64(raw) * 100.0 / float64(maxV)), nil
}

func (b *Backlight) Off() error {
	return writeInt(b.BrightnessPath, 0)
}

func readInt(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(b))
	return strconv.Atoi(s)
}

func writeInt(path string, v int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(v)), 0o644)
}


