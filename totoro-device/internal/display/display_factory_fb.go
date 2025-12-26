//go:build !preview

package display

// NewDisplay 创建显示实例 (Production - Framebuffer)
func NewDisplay(title string, width, height int) (Display, error) {
	disp := &fbDisplay{
		width:  width,
		height: height,
	}
	if err := disp.Init(); err != nil {
		return nil, err
	}
	return disp, nil
}

