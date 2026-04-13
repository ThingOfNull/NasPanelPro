// Package supervisor：kmsg / vcs 监听、TTY 避让与恢复（PRD §5）。
package supervisor

import (
	"context"
	"log"
	"time"

	"naspanel/internal/cfg"
)

// Callbacks 注入 chvt 与渲染挂起。
type Callbacks struct {
	OnYield   func() error
	OnRecover func() error
}

// Supervisor 协调监听 goroutine。
type Supervisor struct {
	cfg cfg.Config
	cb  Callbacks

	mode            Mode
	lastHash        uint64
	vcsStableChange int
	lastActivity    time.Time
	lastChvt        time.Time
}

// New 创建监督器。
func New(c cfg.Config, cb Callbacks) *Supervisor {
	return &Supervisor{
		cfg:          c,
		cb:           cb,
		mode:         ModeGUI,
		lastActivity: time.Now(),
	}
}

// Run 阻塞直到 ctx 取消。
func (s *Supervisor) Run(ctx context.Context) error {
	kmsgCh := make(chan string, 32)
	go kmsgReader(ctx, s.cfg.KMsgPath, kmsgCh)

	vcsp := time.NewTicker(time.Duration(s.cfg.VCSPoll) * time.Millisecond)
	defer vcsp.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, ok := <-kmsgCh:
			if !ok {
				kmsgCh = nil
				continue
			}
			if s.mode == ModeGUI {
				pri := parseKmsgPriority(line)
				if pri >= 0 && pri <= 4 {
					s.yield("kmsg")
				}
			}
		case <-vcsp.C:
			s.tickVCS()
		}
	}
}

func (s *Supervisor) inChvtShield() bool {
	return time.Since(s.lastChvt) < time.Duration(s.cfg.ChvtShield)*time.Millisecond
}

func (s *Supervisor) yield(reason string) {
	if s.cb.OnYield != nil {
		if err := s.cb.OnYield(); err != nil {
			log.Printf("supervisor OnYield: %v", err)
		}
	}
	s.mode = ModeConsole
	s.lastActivity = time.Now()
	s.lastChvt = time.Now()
	s.vcsStableChange = 0
	log.Printf("supervisor: yield (%s) -> console TTY", reason)
}

func (s *Supervisor) recover(reason string) {
	if s.cb.OnRecover != nil {
		if err := s.cb.OnRecover(); err != nil {
			log.Printf("supervisor OnRecover: %v", err)
		}
	}
	s.mode = ModeGUI
	s.lastChvt = time.Now()
	s.vcsStableChange = 0
	log.Printf("supervisor: recover (%s) -> GUI TTY", reason)
}

func (s *Supervisor) tickVCS() {
	if s.inChvtShield() {
		return
	}
	h, err := vcsScreenHash(s.cfg.VCSPath)
	if err != nil {
		return
	}
	if s.mode == ModeGUI {
		if h != s.lastHash {
			s.vcsStableChange++
			if s.vcsStableChange >= s.cfg.VCSConfirmN {
				s.yield("vcs")
			}
		} else {
			s.vcsStableChange = 0
		}
		s.lastHash = h
		return
	}

	// ModeConsole：任何变化视为活动，重置静默计时
	if h != s.lastHash {
		s.lastActivity = time.Now()
		s.vcsStableChange = 0
	}
	s.lastHash = h

	if time.Since(s.lastActivity) >= time.Duration(s.cfg.RecoverSec)*time.Second {
		s.recover("silent")
	}
}
