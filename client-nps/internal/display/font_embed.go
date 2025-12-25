package display

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/golang/freetype/truetype"
)

//go:embed assets/fonts/ArialUnicode.ttf
var arialUnicodeFontData []byte

var (
	fontManager     *FontManager
	fontManagerOnce sync.Once
)

// FontManager 字体管理器
type FontManager struct {
	arialUnicodeRegular *truetype.Font
	arialUnicodeMedium  *truetype.Font
	arialUnicodeBold    *truetype.Font
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
	fmt.Printf("📦 字体数据大小: %.2f MB\n", float64(len(arialUnicodeFontData))/1024/1024)
	
	// 解析 TTF 字体
	font, err := truetype.Parse(arialUnicodeFontData)
	if err != nil {
		fmt.Printf("❌ 字体加载失败: %v\n", err)
		return
	}

	fm.arialUnicodeRegular = font
	fm.arialUnicodeMedium = font
	fm.arialUnicodeBold = font
	fmt.Println("✅ Arial Unicode 字体加载成功！")
}

// GetFont 获取字体
func (fm *FontManager) GetFont(weight FontWeight) *truetype.Font {
	switch weight {
	case FontWeightRegular:
		return fm.arialUnicodeRegular
	case FontWeightMedium:
		return fm.arialUnicodeMedium
	case FontWeightBold:
		return fm.arialUnicodeBold
	default:
		return fm.arialUnicodeRegular
	}
}

// FontWeight 字体粗细
type FontWeight int

const (
	FontWeightRegular FontWeight = iota
	FontWeightMedium
	FontWeightBold
)

