package raypanel

import rl "github.com/gen2brain/raylib-go/raylib"

// blendTwoRenderTextures 在已绑定的渲染目标上绘制两离屏纹理的 alpha 混合。
func blendTwoRenderTextures(t0, t1 rl.Texture2D, lw, lh int32, alpha float32) {
	src := rl.NewRectangle(0, 0, float32(lw), -float32(lh))
	pos := rl.NewVector2(0, 0)
	rl.BeginBlendMode(rl.BlendAlpha)
	rl.DrawTextureRec(t0, src, pos, rl.ColorAlpha(rl.White, 1-alpha))
	rl.DrawTextureRec(t1, src, pos, rl.ColorAlpha(rl.White, alpha))
	rl.EndBlendMode()
}
