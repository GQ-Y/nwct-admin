//go:build linux && !preview

package display

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// 最小可用的 Linux evdev 单指触摸读取器：
// - 兼容 ABS_X/ABS_Y 与 ABS_MT_POSITION_X/Y
// - 用 BTN_TOUCH 或 trackingId 判断按下/抬起
// - 以 SYN_REPORT 作为一帧提交点

type evdevTouch struct {
	fd      int
	devPath string

	screenW int
	screenH int

	absMinX int32
	absMaxX int32
	absMinY int32
	absMaxY int32

	curX int
	curY int

	isDown    bool
	lastDown  bool
	lastX     int
	lastY     int
	hasPos    bool
	hasFrame  bool
	eventsBuf []TouchEvent
}

func newEvdevTouch(screenW, screenH int) *evdevTouch {
	return &evdevTouch{
		fd:      -1,
		screenW: screenW,
		screenH: screenH,
	}
}

func (t *evdevTouch) Init() error {
	path, err := findTouchDevice()
	if err != nil {
		return err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("打开触摸设备失败: %s: %v", path, err)
	}
	t.fd = fd
	t.devPath = path

	// 获取 ABS 范围用于映射到屏幕坐标
	minX, maxX := int32(0), int32(t.screenW-1)
	minY, maxY := int32(0), int32(t.screenH-1)
	if ax, err := ioctlGetAbs(fd, ABS_MT_POSITION_X); err == nil {
		minX, maxX = ax.Minimum, ax.Maximum
	} else if ax, err := ioctlGetAbs(fd, ABS_X); err == nil {
		minX, maxX = ax.Minimum, ax.Maximum
	}
	if ay, err := ioctlGetAbs(fd, ABS_MT_POSITION_Y); err == nil {
		minY, maxY = ay.Minimum, ay.Maximum
	} else if ay, err := ioctlGetAbs(fd, ABS_Y); err == nil {
		minY, maxY = ay.Minimum, ay.Maximum
	}
	if maxX <= minX {
		maxX = minX + 1
	}
	if maxY <= minY {
		maxY = minY + 1
	}
	t.absMinX, t.absMaxX = minX, maxX
	t.absMinY, t.absMaxY = minY, maxY

	return nil
}

func (t *evdevTouch) Close() error {
	if t.fd >= 0 {
		_ = unix.Close(t.fd)
		t.fd = -1
	}
	return nil
}

func (t *evdevTouch) Poll() []TouchEvent {
	if t.fd < 0 {
		return nil
	}
	t.eventsBuf = t.eventsBuf[:0]

	// 读一批 events（非阻塞）
	for {
		var ev inputEvent
		n, err := unix.Read(t.fd, (*(*[unsafe.Sizeof(inputEvent{})]byte)(unsafe.Pointer(&ev)))[:])
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}
			break
		}
		if n != int(unsafe.Sizeof(inputEvent{})) {
			break
		}

		t.handle(ev)
		if t.hasFrame {
			// 一帧结束就输出一次，避免堆积
			out := make([]TouchEvent, len(t.eventsBuf))
			copy(out, t.eventsBuf)
			t.hasFrame = false
			t.eventsBuf = t.eventsBuf[:0]
			return out
		}
	}
	return nil
}

func (t *evdevTouch) handle(ev inputEvent) {
	switch ev.Type {
	case EV_ABS:
		switch ev.Code {
		case ABS_X, ABS_MT_POSITION_X:
			t.curX = t.mapAxis(ev.Value, t.absMinX, t.absMaxX, t.screenW)
			t.hasPos = true
		case ABS_Y, ABS_MT_POSITION_Y:
			t.curY = t.mapAxis(ev.Value, t.absMinY, t.absMaxY, t.screenH)
			t.hasPos = true
		case ABS_MT_TRACKING_ID:
			// -1 表示离开
			if ev.Value < 0 {
				t.isDown = false
			} else {
				t.isDown = true
			}
		}
	case EV_KEY:
		if ev.Code == BTN_TOUCH {
			t.isDown = ev.Value != 0
		}
	case EV_SYN:
		if ev.Code == SYN_REPORT {
			t.emitFrame()
			t.hasFrame = true
		}
	}
}

func (t *evdevTouch) emitFrame() {
	if !t.hasPos && t.isDown == t.lastDown {
		return
	}
	x, y := t.curX, t.curY
	tt := TouchMove
	if t.isDown && !t.lastDown {
		tt = TouchDown
	} else if !t.isDown && t.lastDown {
		tt = TouchUp
	}
	// Move 仅在坐标变化时发送
	if tt == TouchMove && (x == t.lastX && y == t.lastY) {
		return
	}
	t.eventsBuf = append(t.eventsBuf, TouchEvent{Type: tt, X: x, Y: y, Timestamp: 0})
	t.lastDown = t.isDown
	t.lastX, t.lastY = x, y
	t.hasPos = false
}

func (t *evdevTouch) mapAxis(v int32, min, max int32, out int) int {
	if out <= 1 {
		return 0
	}
	if v < min {
		v = min
	}
	if v > max {
		v = max
	}
	num := int64(v - min)
	den := int64(max - min)
	if den <= 0 {
		return 0
	}
	return int(num * int64(out-1) / den)
}

func findTouchDevice() (string, error) {
	cands, _ := filepath.Glob("/dev/input/event*")
	// 优先找名字像 Goodix/GT911 的
	best := ""
	for _, p := range cands {
		name := ""
		if fd, err := unix.Open(p, unix.O_RDONLY|unix.O_NONBLOCK, 0); err == nil {
			if n, e := ioctlGetName(fd); e == nil {
				name = n
			}
			_ = unix.Close(fd)
		}
		low := strings.ToLower(name)
		if strings.Contains(low, "goodix") || strings.Contains(low, "gt911") || strings.Contains(low, "touch") {
			return p, nil
		}
		if best == "" && name != "" {
			best = p
		}
	}
	if best != "" {
		return best, nil
	}
	if len(cands) > 0 {
		return cands[0], nil
	}
	return "", fmt.Errorf("未找到触摸设备（/dev/input/event*）")
}

// ---- linux input/event structs & ioctls ----

type inputEvent struct {
	Time  unix.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type inputAbsInfo struct {
	Value      int32
	Minimum    int32
	Maximum    int32
	Fuzz       int32
	Flat       int32
	Resolution int32
}

const (
	EV_SYN = 0x00
	EV_KEY = 0x01
	EV_ABS = 0x03

	SYN_REPORT = 0

	BTN_TOUCH = 0x014a

	ABS_X              = 0x00
	ABS_Y              = 0x01
	ABS_MT_POSITION_X  = 0x35
	ABS_MT_POSITION_Y  = 0x36
	ABS_MT_TRACKING_ID = 0x39
)

// ioctl helpers
// 这些 ioctl 编码来自 linux/ioctl.h 的宏展开（仅用到少量）
func ioc(dir, typ, nr, size uintptr) uintptr {
	const (
		iocNRBits   = 8
		iocTypeBits = 8
		iocSizeBits = 14
		iocDirBits  = 2

		iocNRShift   = 0
		iocTypeShift = iocNRShift + iocNRBits
		iocSizeShift = iocTypeShift + iocTypeBits
		iocDirShift  = iocSizeShift + iocSizeBits
	)
	return (dir << iocDirShift) | (typ << iocTypeShift) | (nr << iocNRShift) | (size << iocSizeShift)
}

const (
	iocRead = 2
)

func evioCGName(len int) uintptr { return ioc(iocRead, 'E', 0x06, uintptr(len)) }
func evioCGAbs(axis int) uintptr { return ioc(iocRead, 'E', 0x40+uintptr(axis), uintptr(unsafe.Sizeof(inputAbsInfo{}))) }

func ioctlGetName(fd int) (string, error) {
	buf := make([]byte, 256)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), evioCGName(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return "", errno
	}
	// C string
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n]), nil
}

func ioctlGetAbs(fd int, axis int) (*inputAbsInfo, error) {
	var info inputAbsInfo
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), evioCGAbs(axis), uintptr(unsafe.Pointer(&info)))
	if errno != 0 {
		return nil, errno
	}
	return &info, nil
}


