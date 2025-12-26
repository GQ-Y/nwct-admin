package display

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/golang/freetype/truetype"
)

// 优先使用 HarmonyOS Sans（简体中文），回退到 Arial Unicode
//go:embed assets/fonts/HarmonyOS_Sans_SC_Regular.ttf
var harmonySansRegularData []byte

//go:embed assets/fonts/HarmonyOS_Sans_SC_Medium.ttf
var harmonySansMediumData []byte

//go:embed assets/fonts/HarmonyOS_Sans_SC_Bold.ttf
var harmonySansBoldData []byte

//go:embed assets/fonts/ArialUnicode.ttf
var arialUnicodeFontData []byte

var (
	fontManager     *FontManager
	fontManagerOnce sync.Once
)

// FontManager 字体管理器
type FontManager struct {
	regular *truetype.Font
	medium  *truetype.Font
	bold    *truetype.Font
}

// GetFontManager 获取字体管理器单例
func GetFontManager() *FontManager {
	fontManagerOnce.Do(func() {
		fontManager = &FontManager{}
		fontManager.loadFonts()
	})
	return fontManager
}

// loadFonts 加载字体
func (fm *FontManager) loadFonts() {
	// 1) HarmonyOS Sans（SC）
	ok := true
	if f, err := truetype.Parse(harmonySansRegularData); err == nil {
		fm.regular = f
	} else {
		ok = false
		fmt.Printf("❌ HarmonyOS Sans Regular 加载失败: %v\n", err)
	}
	if f, err := truetype.Parse(harmonySansMediumData); err == nil {
		fm.medium = f
	} else {
		ok = false
		fmt.Printf("❌ HarmonyOS Sans Medium 加载失败: %v\n", err)
	}
	if f, err := truetype.Parse(harmonySansBoldData); err == nil {
		fm.bold = f
	} else {
		ok = false
		fmt.Printf("❌ HarmonyOS Sans Bold 加载失败: %v\n", err)
	}
	if ok && fm.regular != nil && fm.medium != nil && fm.bold != nil {
		fmt.Println("✅ HarmonyOS Sans（SC）字体加载成功！")
		return
	}

	// 2) 回退：Arial Unicode
	fmt.Printf("📦 回退字体数据大小: %.2f MB\n", float64(len(arialUnicodeFontData))/1024/1024)
	if font, err := truetype.Parse(arialUnicodeFontData); err == nil {
		fm.regular = font
		fm.medium = font
		fm.bold = font
		fmt.Println("✅ Arial Unicode 字体加载成功（回退）！")
		return
	}

	fmt.Println("❌ 字体加载失败：无可用内置字体")
}

// GetFont 获取字体
func (fm *FontManager) GetFont(weight FontWeight) *truetype.Font {
	switch weight {
	case FontWeightRegular:
		return fm.regular
	case FontWeightMedium:
		return fm.medium
	case FontWeightBold:
		return fm.bold
	default:
		return fm.regular
	}
}

// FontWeight 字体粗细
type FontWeight int

const (
	FontWeightRegular FontWeight = iota
	FontWeightMedium
	FontWeightBold
)

