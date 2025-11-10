package main

import "sync/atomic"

type LamportClock struct {
	time int64
}

func (lc *LamportClock) Tick() int64 {
	return atomic.AddInt64(&lc.time, 1)
}

func (lc *LamportClock) Receive(remote int64) int64 {
	for {
		old := atomic.LoadInt64(&lc.time)
		newVal := max(old, remote) + 1
		if atomic.CompareAndSwapInt64(&lc.time, old, newVal) {
			return newVal
		}
	}
}

func (lc *LamportClock) Time() int64 {
	return atomic.LoadInt64(&lc.time)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
