package supervisor

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

// parseKmsgPriority 解析内核 printk 风格 `<12>message`，返回优先级数字（无法解析则 -1）。
func parseKmsgPriority(line string) int {
	i := strings.Index(line, "<")
	if i < 0 {
		return -1
	}
	j := strings.Index(line[i+1:], ">")
	if j < 0 {
		return -1
	}
	j += i + 1
	n, err := strconv.Atoi(strings.TrimSpace(line[i+1 : j]))
	if err != nil {
		return -1
	}
	return n
}

// kmsgReader 从 /dev/kmsg 读行，发往 ch；ctx 取消时退出（goroutine 可取消）。
func kmsgReader(ctx context.Context, path string, ch chan<- string) {
	defer close(ch)
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for {
		// 设置短超时，使阻塞读可被 ctx 取消打断。
		_ = f.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if sc.Scan() {
			select {
			case <-ctx.Done():
				return
			case ch <- sc.Text():
			}
			continue
		}
		// Scan 返回 false：可能是超时（os.ErrDeadlineExceeded），也可能是真正的 EOF/错误。
		if ctx.Err() != nil {
			return
		}
		// 超时是正常情况，重置 scanner 错误后继续。
		if err := sc.Err(); err != nil {
			// 使用新的 Scanner 继续（超时后原 scanner 状态已损坏）。
			sc = bufio.NewScanner(f)
		}
	}
}
