package core

// NormalizeRotationDegrees 将角度归一到 0/90/180/270。
func NormalizeRotationDegrees(d int) int {
	d %= 360
	if d < 0 {
		d += 360
	}
	switch d {
	case 0, 90, 180, 270:
		return d
	default:
		return 0
	}
}
