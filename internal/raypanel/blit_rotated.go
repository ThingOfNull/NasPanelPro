package raypanel

import rl "github.com/gen2brain/raylib-go/raylib"

// blitRotatedCamera 用 2D 相机旋转离屏纹理到物理屏。
func blitRotatedCamera(rt rl.RenderTexture2D, rot int, sw, sh, lw, lh int32) {
	tex := rt.Texture
	src := rl.NewRectangle(0, 0, float32(lw), -float32(lh))
	var rotDeg float32
	switch rot {
	case 90:
		rotDeg = 90
	case 270:
		rotDeg = -90
	case 180:
		rotDeg = 180
	default:
		rotDeg = 0
	}
	cam := rl.Camera2D{
		Target:   rl.NewVector2(float32(lw)/2, float32(lh)/2),
		Offset:   rl.NewVector2(float32(sw)/2, float32(sh)/2),
		Rotation: rotDeg,
		Zoom:     1,
	}
	rl.BeginMode2D(cam)
	rl.DrawTextureRec(tex, src, rl.NewVector2(0, 0), tintWhite)
	rl.EndMode2D()
}

// blitRotatedPro 可选路径（环境变量 NASPANEL_DRM_BLIT_PRO=1）。
func blitRotatedPro(rt rl.RenderTexture2D, rot int, sw, sh, lw, lh int32) {
	tex := rt.Texture
	src := rl.NewRectangle(0, 0, float32(lw), -float32(lh))
	var rotDeg float32
	switch rot {
	case 90:
		rotDeg = 90
	case 270:
		rotDeg = 270
	case 180:
		rotDeg = 180
	default:
		rotDeg = 0
	}
	dest := rl.NewRectangle(float32(sw)/2, float32(sh)/2, float32(lw), float32(lh))
	origin := rl.NewVector2(float32(lw)/2, float32(lh)/2)
	rl.DrawTexturePro(tex, src, dest, origin, rotDeg, rl.White)
}
