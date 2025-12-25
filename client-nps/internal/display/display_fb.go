//go:build !preview

package display

import (
	"fmt"
	"image"
	"os"
	"syscall"
	"unsafe"
)

type fbDisplay struct {
	fbFile     *os.File
	fbMem      []byte
	width      int
	height     int
	backBuffer *image.RGBA
}

type fbVarScreenInfo struct {
	xres           uint32
	yres           uint32
	xres_virtual   uint32
	yres_virtual   uint32
	xoffset        uint32
	yoffset        uint32
	bits_per_pixel uint32
	grayscale      uint32
	// 还有更多字段，但我们只需要这些
}

const (
	FBIOGET_VSCREENINFO = 0x4600
)

func newDisplay() Display {
	return &fbDisplay{
		width:  480,
		height: 480,
	}
}

func (d *fbDisplay) Init() error {
	// 打开 framebuffer 设备
	fbFile, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("打开 /dev/fb0 失败: %v", err)
	}
	d.fbFile = fbFile

	// 获取 framebuffer 信息
	var fbInfo fbVarScreenInfo
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fbFile.Fd()),
		uintptr(FBIOGET_VSCREENINFO),
		uintptr(unsafe.Pointer(&fbInfo)),
	)
	if errno != 0 {
		return fmt.Errorf("获取 framebuffer 信息失败: %v", errno)
	}

	d.width = int(fbInfo.xres)
	d.height = int(fbInfo.yres)

	// 映射 framebuffer 内存
	fbSize := int(fbInfo.xres * fbInfo.yres * fbInfo.bits_per_pixel / 8)
	fbMem, err := syscall.Mmap(
		int(fbFile.Fd()),
		0,
		fbSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("映射 framebuffer 内存失败: %v", err)
	}
	d.fbMem = fbMem

	// 创建离屏缓冲区
	d.backBuffer = image.NewRGBA(image.Rect(0, 0, d.width, d.height))

	return nil
}

func (d *fbDisplay) Close() error {
	if d.fbMem != nil {
		syscall.Munmap(d.fbMem)
	}
	if d.fbFile != nil {
		d.fbFile.Close()
	}
	return nil
}

func (d *fbDisplay) GetWidth() int {
	return d.width
}

func (d *fbDisplay) GetHeight() int {
	return d.height
}

func (d *fbDisplay) GetBackBuffer() *image.RGBA {
	return d.backBuffer
}

func (d *fbDisplay) Update() error {
	// 将 backBuffer 复制到 framebuffer 内存
	copy(d.fbMem, d.backBuffer.Pix)
	return nil
}

func (d *fbDisplay) PollEvents() (shouldQuit bool) {
	// 生产环境暂不处理退出事件
	return false
}

// GetTouchEvents 获取触摸事件
func (d *fbDisplay) GetTouchEvents() []TouchEvent {
	// TODO: 实现 GT911 evdev 触摸事件读取
	return nil
}

