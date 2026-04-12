package supervisor

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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

// kmsgReader 从 /dev/kmsg 读行，发往 ch（阻塞读在独立 goroutine）。
func kmsgReader(path string, ch chan<- string) {
	defer close(ch)
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ch <- sc.Text()
	}
}
