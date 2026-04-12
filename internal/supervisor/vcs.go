package supervisor

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
)

// vcsScreenHash 计算 vcs 缓冲区内容 hash（整屏，简化实现）。
func vcsScreenHash(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, nil
	}
	h := sha256.Sum256(b)
	return binary.BigEndian.Uint64(h[:8]), nil
}
