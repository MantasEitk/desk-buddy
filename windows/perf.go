package main

import (
	"sync"
	"time"
)

type perfNow struct {
	CPU       float64 `json:"cpu"`        // % of one core, over the last minute (app + server)
	Mem       float64 `json:"mem"`        // MB, app + server
	ServerMem float64 `json:"server_mem"` // MB of that for the server ("On this PC"), else 0
	Warming   bool    `json:"warming"`    // the first 90 s after starting are busy
	Ready     bool    `json:"ready"`
}

var (
	perfMu  sync.Mutex
	perf    perfNow
	started = time.Now()
)

// measure samples every 10 s (cheap: two system calls per process).
func measure() {
	type sample struct {
		at  time.Time
		cpu float64
	}
	var samples []sample
	for {
		cpu, mem, _ := usage(0)
		var smem float64
		srvMu.Lock()
		var pid int
		if srv != nil && srv.Process != nil {
			pid = srv.Process.Pid
		}
		srvMu.Unlock()
		if pid != 0 {
			if c, m, ok := usage(pid); ok {
				cpu += c
				smem = m
				mem += m
			}
		}
		now := time.Now()
		samples = append(samples, sample{now, cpu})
		for len(samples) > 1 && now.Sub(samples[0].at) > 65*time.Second {
			samples = samples[1:]
		}
		p := perfNow{Mem: mem, ServerMem: smem, Warming: now.Sub(started) < 90*time.Second}
		if first := samples[0]; now.Sub(first.at) >= 9*time.Second {
			p.CPU = (cpu - first.cpu) / now.Sub(first.at).Seconds() * 100
			p.Ready = true
		}
		perfMu.Lock()
		perf = p
		perfMu.Unlock()
		time.Sleep(10 * time.Second)
	}
}

func perfInfo() perfNow {
	perfMu.Lock()
	defer perfMu.Unlock()
	return perf
}
