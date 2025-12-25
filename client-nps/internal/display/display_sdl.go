//go:build preview

package display

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

type sdlDisplay struct {
	window      *sdl.Window
	renderer    *sdl.Renderer
	texture     *sdl.Texture
	width       int
	height      int
	backBuffer  *image.RGBA
	touchEvents []TouchEvent
}

// NewSDL2 创建 SDL2 显示
func NewSDL2(title string, width, height int) Display {
	return &sdlDisplay{
		width:  width,
		height: height,
	}
}

func (d *sdlDisplay) Init() error {
	// 初始化 SDL
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return fmt.Errorf("SDL 初始化失败: %v", err)
	}

	// 创建窗口
	window, err := sdl.CreateWindow(
		"NWCT Display Preview - 480x480",
		sdl.WINDOWPOS_CENTERED,
		sdl.WINDOWPOS_CENTERED,
		int32(d.width),
		int32(d.height),
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		return fmt.Errorf("创建窗口失败: %v", err)
	}
	d.window = window

	// 创建渲染器
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		return fmt.Errorf("创建渲染器失败: %v", err)
	}
	d.renderer = renderer

	// 创建纹理
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_ABGR8888,
		sdl.TEXTUREACCESS_STREAMING,
		int32(d.width),
		int32(d.height),
	)
	if err != nil {
		return fmt.Errorf("创建纹理失败: %v", err)
	}
	d.texture = texture

	// 创建离屏缓冲区
	d.backBuffer = image.NewRGBA(image.Rect(0, 0, d.width, d.height))

	return nil
}

func (d *sdlDisplay) Close() error {
	if d.texture != nil {
		d.texture.Destroy()
	}
	if d.renderer != nil {
		d.renderer.Destroy()
	}
	if d.window != nil {
		d.window.Destroy()
	}
	sdl.Quit()
	return nil
}

func (d *sdlDisplay) GetWidth() int {
	return d.width
}

func (d *sdlDisplay) GetHeight() int {
	return d.height
}

func (d *sdlDisplay) GetBackBuffer() *image.RGBA {
	return d.backBuffer
}

func (d *sdlDisplay) Update() error {
	// 将 backBuffer 复制到纹理（使用 unsafe.Pointer）
	pitch := d.backBuffer.Stride
	rect := &sdl.Rect{X: 0, Y: 0, W: int32(d.width), H: int32(d.height)}
	
	if err := d.texture.Update(rect, unsafe.Pointer(&d.backBuffer.Pix[0]), pitch); err != nil {
		return fmt.Errorf("更新纹理失败: %v", err)
	}

	// 渲染纹理到窗口
	d.renderer.Clear()
	d.renderer.Copy(d.texture, nil, nil)
	d.renderer.Present()

	return nil
}

func (d *sdlDisplay) PollEvents() (shouldQuit bool) {
	// 清空上一帧的触摸事件
	d.touchEvents = nil
	
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			return true
		case *sdl.KeyboardEvent:
			if e.Type == sdl.KEYDOWN && e.Keysym.Sym == sdl.K_ESCAPE {
				return true
			}
		case *sdl.MouseButtonEvent:
			// 处理鼠标点击事件
			d.handleMouseEvent(e)
		}
	}
	return false
}

// handleMouseEvent 处理鼠标事件
func (d *sdlDisplay) handleMouseEvent(e *sdl.MouseButtonEvent) {
	var touchType TouchType
	if e.Type == sdl.MOUSEBUTTONDOWN {
		touchType = TouchDown
	} else if e.Type == sdl.MOUSEBUTTONUP {
		touchType = TouchUp
	} else {
		return
	}
	
	d.touchEvents = append(d.touchEvents, TouchEvent{
		Type:      touchType,
		X:         int(e.X),
		Y:         int(e.Y),
		Timestamp: int64(e.Timestamp),
	})
}

// GetTouchEvents 获取触摸事件
func (d *sdlDisplay) GetTouchEvents() []TouchEvent {
	return d.touchEvents
}

