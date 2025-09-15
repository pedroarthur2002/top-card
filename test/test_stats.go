package test

import "sync"

type TestStats struct {
    Success int
    Failed  int
    Errors  int
    mu      sync.Mutex
}

func (s *TestStats) AddSuccess() {
    s.mu.Lock()
    s.Success++
    s.mu.Unlock()
}

func (s *TestStats) AddFailed() {
    s.mu.Lock()
    s.Failed++
    s.mu.Unlock()
}

func (s *TestStats) AddError() {
    s.mu.Lock()
    s.Errors++
    s.mu.Unlock()
}