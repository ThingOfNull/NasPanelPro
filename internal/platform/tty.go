// Package platform 封装与 Linux TTY、chvt 相关的宿主机能力，便于单元测试时用 fake 替换。
package platform

import (
	"fmt"
	"os/exec"
)

// Chvt 切换到指定虚拟终端（需 CAP_SYS_TTY_CONFIG 或等价权限）。
func Chvt(n int) error {
	path, err := exec.LookPath("chvt")
	if err != nil {
		return fmt.Errorf("chvt: %w", err)
	}
	cmd := exec.Command(path, fmt.Sprintf("%d", n))
	return cmd.Run()
}
