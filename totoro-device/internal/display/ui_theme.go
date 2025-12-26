package display

import "image/color"

// 鸿蒙浅色系配色规范
var (
	// 背景色
	ColorBackgroundStart = color.RGBA{255, 255, 255, 255} // 纯白
	ColorBackgroundEnd   = color.RGBA{241, 245, 249, 255} // 极浅灰蓝

	// 文字颜色
	ColorTextPrimary   = color.RGBA{30, 41, 59, 255}   // 深黑 (标题)
	ColorTextSecondary = color.RGBA{100, 116, 139, 255} // 深灰 (副标题)
	ColorTextLight     = color.RGBA{148, 163, 184, 255} // 浅灰 (提示/标签)

	// 功能色
	ColorBrandBlue    = color.RGBA{59, 130, 246, 255}  // 品牌蓝
	ColorSuccessGreen = color.RGBA{16, 185, 129, 255} // 成功绿
	ColorWarningOrange = color.RGBA{245, 158, 11, 255} // 警告橙
	ColorErrorRed     = color.RGBA{239, 68, 68, 255}   // 错误红
	ColorPurple       = color.RGBA{139, 92, 246, 255}  // 装饰紫

	// 交互色
	ColorSeparator  = color.RGBA{226, 232, 240, 255} // 分隔线
	ColorPressed    = color.RGBA{241, 245, 249, 255} // 按下背景
	ColorOverlay    = color.RGBA{0, 0, 0, 100}       // 遮罩层
	ColorKeyboardBg = color.RGBA{203, 213, 225, 255} // 键盘背景
	ColorKeyBg      = color.RGBA{255, 255, 255, 255} // 按键背景
)

