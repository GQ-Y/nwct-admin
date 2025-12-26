//go:build preview

package display

// NewDisplay 创建显示实例 (Preview)
func NewDisplay(title string, width, height int) (Display, error) {
	disp := NewSDL2(title, width, height)
	if err := disp.Init(); err != nil {
		return nil, err
	}
	return disp, nil
}

