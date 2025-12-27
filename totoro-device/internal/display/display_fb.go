//go:build !preview

package display

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"syscall"
	"unsafe"

	"totoro-device/internal/logger"
)

type fbDisplay struct {
	fbFile *os.File
	fbMem  []byte
	// width/height: UI 后缓冲的逻辑分辨率（可为 720x720）
	width  int
	height int
	// fbWidth/fbHeight: /dev/fb0 的真实分辨率
	fbWidth    int
	fbHeight   int
	fbBpp      int
	backBuffer *image.RGBA

	touch touchReader
}

// fbVarScreenInfoRaw:
// Linux 的 FBIOGET_VSCREENINFO 会向用户态写入完整的 struct fb_var_screeninfo。
// 若这里定义的结构体过小，会导致内核写越界 -> 运行时崩溃（arm 上更明显）。
//
// 为避免与不同内核版本的字段差异/对齐问题，这里用足够大的原始 buffer 接收，
// 再解析我们关心的字段：
// - xres/yres/xres_virtual/yres_virtual/xoffset/yoffset/bits_per_pixel/grayscale
type fbVarScreenInfoRaw [160]byte

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
	var fbInfo fbVarScreenInfoRaw
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fbFile.Fd()),
		uintptr(FBIOGET_VSCREENINFO),
		uintptr(unsafe.Pointer(&fbInfo[0])),
	)
	if errno != 0 {
		return fmt.Errorf("获取 framebuffer 信息失败: %v", errno)
	}

	// Linux fb_var_screeninfo 为小端；Luckfox Pico Ultra (armv7l) 也是小端。
	d.fbWidth = int(binary.LittleEndian.Uint32(fbInfo[0:4]))
	d.fbHeight = int(binary.LittleEndian.Uint32(fbInfo[4:8]))
	d.fbBpp = int(binary.LittleEndian.Uint32(fbInfo[24:28]))

	// 如果外部没有指定逻辑分辨率，则跟随真实 framebuffer
	if d.width <= 0 || d.height <= 0 {
		d.width = d.fbWidth
		d.height = d.fbHeight
	}

	// 映射 framebuffer 内存
	fbSize := int(uint64(d.fbWidth) * uint64(d.fbHeight) * uint64(d.fbBpp) / 8)
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
		if err := d.touch.Init(); err != nil {
			logger.Warn("触摸初始化失败: %v", err)
		} else {
			logger.Info("触摸已启用（evdev）")
		}
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
