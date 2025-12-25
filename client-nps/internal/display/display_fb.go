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
	// width/height: UI 后缓冲的逻辑分辨率（可为 720x720）
	width  int
	height int
	// fbWidth/fbHeight: /dev/fb0 的真实分辨率
	fbWidth  int
	fbHeight int
	fbBpp    int
	backBuffer *image.RGBA

	touch touchReader
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

	d.fbWidth = int(fbInfo.xres)
	d.fbHeight = int(fbInfo.yres)
	d.fbBpp = int(fbInfo.bits_per_pixel)

	// 如果外部没有指定逻辑分辨率，则跟随真实 framebuffer
	if d.width <= 0 || d.height <= 0 {
		d.width = d.fbWidth
		d.height = d.fbHeight
	}

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

	// 初始化触摸（linux evdev）
	d.touch = newLinuxEvdevTouch(d.width, d.height)
	if d.touch != nil {
		_ = d.touch.Init()
	}

	return nil
}

func (d *fbDisplay) Close() error {
	if d.touch != nil {
		_ = d.touch.Close()
	}
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
	// 将 backBuffer 刷新到 framebuffer
	// - 如果逻辑分辨率与 fb 一致：直接 memcpy
	// - 否则：做一次缩放（nearest），保证 Linux 端也可用 720x720 逻辑 UI
	if d.fbWidth == d.width && d.fbHeight == d.height && d.fbBpp == 32 {
		copy(d.fbMem, d.backBuffer.Pix)
		return nil
	}

	// 仅处理 32bpp 的缩放路径
	if d.fbBpp != 32 || d.fbWidth <= 0 || d.fbHeight <= 0 || len(d.fbMem) < d.fbWidth*d.fbHeight*4 {
		// 兜底：尽量 copy 前半段，避免 panic
		n := len(d.fbMem)
		if len(d.backBuffer.Pix) < n {
			n = len(d.backBuffer.Pix)
		}
		if n > 0 {
			copy(d.fbMem[:n], d.backBuffer.Pix[:n])
		}
		return nil
	}

	srcW := d.width
	srcH := d.height
	dstW := d.fbWidth
	dstH := d.fbHeight
	if srcW <= 0 || srcH <= 0 {
		return nil
	}

	// nearest neighbor
	for dy := 0; dy < dstH; dy++ {
		sy := dy * srcH / dstH
		dstRow := dy * dstW * 4
		srcRow := sy * d.backBuffer.Stride
		for dx := 0; dx < dstW; dx++ {
			sx := dx * srcW / dstW
			si := srcRow + sx*4
			di := dstRow + dx*4
			// RGBA 原样写入（假设 fb0 使用 32bpp 且与现有实现一致）
			d.fbMem[di+0] = d.backBuffer.Pix[si+0]
			d.fbMem[di+1] = d.backBuffer.Pix[si+1]
			d.fbMem[di+2] = d.backBuffer.Pix[si+2]
			d.fbMem[di+3] = d.backBuffer.Pix[si+3]
		}
	}
	return nil
}

func (d *fbDisplay) PollEvents() (shouldQuit bool) {
	// 生产环境暂不处理退出事件
	return false
}

// GetTouchEvents 获取触摸事件
func (d *fbDisplay) GetTouchEvents() []TouchEvent {
	if d.touch == nil {
		return nil
	}
	return d.touch.Poll()
}

