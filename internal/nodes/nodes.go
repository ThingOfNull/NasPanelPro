// Package nodes 管理 Netdata 节点配置（configs/nodes.json）。
package nodes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

// Node 单个 Netdata 接入点。
type Node struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	APIKey string `json:"api_key,omitempty"`
	// Secure 为 true 且 Host 未写 scheme 时使用 https://（常见于自定义 TLS 端口如 20489）
	Secure bool `json:"secure,omitempty"`
}

// BaseURL 返回 http(s)://host:port（不含尾斜杠）。
func (n *Node) BaseURL() string {
	h := strings.TrimSpace(n.Host)
	if h == "" {
		return ""
	}
	if strings.HasPrefix(h, "http://") || strings.HasPrefix(h, "https://") {
		u, err := url.Parse(h)
		if err != nil {
			return strings.TrimRight(h, "/")
		}
		port := n.Port
		if port > 0 && u.Port() == "" {
			u.Host = fmt.Sprintf("%s:%d", u.Hostname(), port)
		}
		return strings.TrimRight(u.String(), "/")
	}
	port := n.Port
	if port <= 0 {
		port = 19999
	}
	scheme := "http"
	if n.Secure {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, h, port)
}

// File 持久化文件内容。
type File struct {
	Nodes []Node `json:"nodes"`
}

// Validate 校验节点 id 唯一、必填字段。
func (f *File) Validate() error {
	if f == nil {
		return errors.New("nil nodes file")
	}
	seen := make(map[string]struct{})
	for i, n := range f.Nodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			return fmt.Errorf("nodes[%d]: empty id", i)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate node id %q", id)
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(n.Host) == "" {
			return fmt.Errorf("node %q: empty host", id)
		}
		if n.Port < 0 {
			return fmt.Errorf("node %q: invalid port", id)
		}
	}
	return nil
}

// LoadFile 读取 JSON。
func LoadFile(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	if err := f.Validate(); err != nil {
		return File{}, err
	}
	return f, nil
}

// SaveFile 原子写入。
func SaveFile(path string, f File) error {
	if err := f.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// DefaultFile 空列表占位。
func DefaultFile() File {
	return File{Nodes: []Node{}}
}

// ByID 查找节点。
func (f *File) ByID(id string) (Node, bool) {
	id = strings.TrimSpace(id)
	for _, n := range f.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// First 返回节点列表中的第一个条目（文件内顺序）。无节点时 ok 为 false。
func (f *File) First() (Node, bool) {
	if f == nil || len(f.Nodes) == 0 {
		return Node{}, false
	}
	return f.Nodes[0], true
}

// Store 并发安全的节点快照（供 HTTP 与 Poller 读取）。
type Store struct {
	v atomic.Value // *File
}

func (s *Store) Get() File {
	x := s.v.Load()
	if x == nil {
		return DefaultFile()
	}
	src := x.(*File)
	cp := *src
	cp.Nodes = append([]Node(nil), src.Nodes...)
	return cp
}

func (s *Store) Put(f File) {
	cp := f
	cp.Nodes = append([]Node(nil), f.Nodes...)
	s.v.Store(&cp)
}

func (s *Store) Ptr() *File {
	x := s.v.Load()
	if x == nil {
		d := DefaultFile()
		return &d
	}
	return x.(*File)
}
