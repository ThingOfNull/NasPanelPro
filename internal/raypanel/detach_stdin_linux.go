//go:build linux

package raypanel

import (
	"log"
	"os"

	"golang.org/x/sys/unix"
)

// stdinTTYDup 在分离 fd0 之前 dup 出的原 stdin（多为 SSH PTY slave），进程存活期间保持打开。
// 若直接让 fd0 指向 /dev/null 并关掉 PTY 上最后一处引用，进程常失去与控制终端的关联，
// 终端上按 Ctrl+C 时内核不会把 SIGINT 发给本进程（仅有 ^C 回显，Go 侧永远收不到）。
var stdinTTYDup = -1

// detachStdinFromTTY 在 InitWindow 之前把 fd0 接到 /dev/null，避免 C 侧乱改 SSH 行规程。
// 通过 stdinTTYDup 继续持有 PTY 引用，尽量保留键盘 SIGINT 递达。
// 完全不要分离时设置 NASPANEL_DRM_KEEP_STDIN=1。
func detachStdinFromTTY() {
	if os.Getenv("NASPANEL_DRM_KEEP_STDIN") == "1" {
		return
	}

	if backup, err := unix.Dup(unix.Stdin); err == nil {
		stdinTTYDup = backup
	} else {
		log.Printf("raypanel: Dup(stdin) failed, still redirecting fd0 to /dev/null (Ctrl+C may not work): %v", err)
	}

	nf, err := os.OpenFile("/dev/null", os.O_RDONLY, 0)
	if err != nil {
		if stdinTTYDup >= 0 {
			_ = unix.Close(stdinTTYDup)
			stdinTTYDup = -1
		}
		log.Printf("raypanel: open /dev/null failed, skip stdin detach: %v", err)
		return
	}
	defer nf.Close()

	if err := unix.Dup2(int(nf.Fd()), unix.Stdin); err != nil {
		if stdinTTYDup >= 0 {
			_ = unix.Close(stdinTTYDup)
			stdinTTYDup = -1
		}
		log.Printf("raypanel: Dup2 stdin to /dev/null: %v", err)
	}
}
