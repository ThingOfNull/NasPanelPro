package raypanel

import (
	"image/color"
	"log"
	"os"
	"strconv"
	"strings"

	"naspanel/internal/layout"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	panelUIFont   rl.Font
	panelUIFontOk bool
)

// 布局里未出现时的补充字符（标点与常见词），减轻缺字。
const uiFontExtraRunes = ` ，。！？【】（）《》、；：·…—\n\t` +
	`年月日时分秒周` +
	`一二三四五六七八九十百千万亿零` +
	`字节网络内存磁盘温度负载百分比` +
	`系统用户空闲可用已用读写收发` +
	`上下出入错误警告失败成功在线`

func unloadPanelUIFont() {
	if !panelUIFontOk {
		return
	}
	rl.UnloadFont(panelUIFont)
	panelUIFont = rl.Font{}
	panelUIFontOk = false
}

func maxFontGlyphs() int {
	n := 4096
	if s := os.Getenv("NASPANEL_FONT_MAX_GLYPHS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 64 && v <= 20000 {
			n = v
		}
	}
	return n
}

func buildCodepointRunes(lc *layout.LayoutConfig, maxGlyphs int) []rune {
	seen := make(map[rune]struct{}, maxGlyphs)
	out := make([]rune, 0, maxGlyphs)
	add := func(r rune) {
		if len(out) >= maxGlyphs {
			return
		}
		if r < 32 && r != '\n' && r != '\t' {
			return
		}
		if _, ok := seen[r]; ok {
			return
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	for r := rune(32); r < 127; r++ {
		add(r)
	}
	if lc != nil {
		for _, sc := range lc.Scenes {
			for _, r := range sc.Name {
				add(r)
			}
			for i := range sc.Widgets {
				w := &sc.Widgets[i]
				for _, r := range w.Label {
					add(r)
				}
				for _, r := range w.Format {
					add(r)
				}
				for _, r := range w.ValueExpr {
					add(r)
				}
			}
		}
	}
	for _, r := range uiFontExtraRunes {
		add(r)
	}
	return out
}

func fontPathCandidates() []string {
	if p := strings.TrimSpace(os.Getenv("NASPANEL_FONT")); p != "" {
		return []string{p}
	}
	return []string{
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-VF.ttf.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
		"/usr/share/fonts/truetype/arphic/uming.ttc",
		"/usr/share/fonts/TTF/NotoSansCJK-Regular.ttc",
	}
}

func findUIFontPath() string {
	for _, p := range fontPathCandidates() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// InitPanelUIFont 在 InitWindow 之后调用；从 NASPANEL_FONT 或常见系统路径加载 TTF/TTC。
func InitPanelUIFont(lc *layout.LayoutConfig) {
	unloadPanelUIFont()
	path := findUIFontPath()
	if path == "" {
		log.Printf("raypanel: no font file found (set NASPANEL_FONT to a .ttf/.ttc with CJK); UI falls back to ASCII")
		return
	}
	maxG := maxFontGlyphs()
	cps := buildCodepointRunes(lc, maxG)
	baseSize := int32(32)
	f := rl.LoadFontEx(path, baseSize, cps)
	if !rl.IsFontValid(f) && len(cps) > 1500 {
		log.Printf("raypanel: font atlas failed with %d glyphs, retrying 1500", len(cps))
		cps = buildCodepointRunes(lc, 1500)
		f = rl.LoadFontEx(path, baseSize, cps)
	}
	if !rl.IsFontValid(f) {
		log.Printf("raypanel: LoadFontEx failed for %s", path)
		return
	}
	rl.SetTextureFilter(f.Texture, rl.FilterBilinear)
	panelUIFont = f
	panelUIFontOk = true
	log.Printf("raypanel: UI font %s (%d codepoints, base %dpx)", path, len(cps), baseSize)
}

func drawUIText(font rl.Font, ok bool, text string, x, y float32, size float32, col color.RGBA) {
	if text == "" {
		return
	}
	if ok && rl.IsFontValid(font) {
		rl.DrawTextEx(font, text, rl.NewVector2(x, y), size, 1, col)
		return
	}
	rl.DrawText(text, int32(x), int32(y), int32(size+0.5), col)
}

func drawUIMultiline(font rl.Font, ok bool, text string, x, y, size float32, col color.RGBA) {
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	lineH := size * 1.22
	for i, ln := range lines {
		drawUIText(font, ok, ln, x, y+float32(i)*lineH, size, col)
	}
}
