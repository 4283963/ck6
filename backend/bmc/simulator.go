package bmc

import (
	"math/rand"
	"sync"
	"time"
)

type ServerStatus struct {
	ID          string  `json:"id"`
	FanSpeed    int     `json:"fan_speed"`
	CPUTemp     float64 `json:"cpu_temp"`
	PowerUsage  float64 `json:"power_usage"`
	PowerLimit  float64 `json:"power_limit"`
	LastUpdated time.Time `json:"last_updated"`
}

type BMCSimulator struct {
	servers map[string]*ServerStatus
	mu      sync.RWMutex
}

func NewBMCSimulator(count int) *BMCSimulator {
	sim := &BMCSimulator{
		servers: make(map[string]*ServerStatus),
	}

	for i := 0; i < count; i++ {
		id := serverID(i)
		baseTemp := 45.0 + rand.Float64()*15.0
		sim.servers[id] = &ServerStatus{
			ID:         id,
			FanSpeed:   3000 + rand.Intn(3000),
			CPUTemp:    baseTemp,
			PowerUsage: 200.0 + rand.Float64()*150.0,
			PowerLimit: 500.0,
			LastUpdated: time.Now(),
		}
	}

	return sim
}

func serverID(index int) string {
	rack := index/10 + 1
	slot := index%10 + 1
	return string(rune('A'+rack-1)) + formatSlot(slot)
}

func formatSlot(slot int) string {
	if slot < 10 {
		return "0" + string(rune('0'+slot))
	}
	return string(rune('0'+slot/10)) + string(rune('0'+slot%10))
}

func (s *BMCSimulator) GetStatus(id string) (ServerStatus, bool) {
	if s == nil || s.servers == nil {
		return ServerStatus{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	svr, ok := s.servers[id]
	if !ok || svr == nil {
		return ServerStatus{}, false
	}
	return *svr, true
}

func (s *BMCSimulator) ListAll() []ServerStatus {
	if s == nil || s.servers == nil {
		return []ServerStatus{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ServerStatus, 0, len(s.servers))
	for _, svr := range s.servers {
		if svr != nil {
			result = append(result, *svr)
		}
	}
	return result
}

func (s *BMCSimulator) SetPowerLimit(id string, limit float64) bool {
	if s == nil || s.servers == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	svr, ok := s.servers[id]
	if !ok || svr == nil {
		return false
	}
	if limit < 100 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	svr.PowerLimit = limit
	return true
}

func (s *BMCSimulator) Tick() {
	if s == nil || s.servers == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, svr := range s.servers {
		if svr == nil {
			continue
		}
		if svr.PowerLimit <= 0 {
			svr.PowerLimit = 500.0
		}
		tempDrift := (rand.Float64() - 0.5) * 3.0
		targetTemp := 40.0 + (svr.PowerUsage / svr.PowerLimit) * 40.0
		svr.CPUTemp = svr.CPUTemp*0.9 + targetTemp*0.1 + tempDrift

		if svr.CPUTemp > 70.0 {
			svr.FanSpeed = min(8000, svr.FanSpeed+200+rand.Intn(300))
		} else if svr.CPUTemp < 50.0 {
			svr.FanSpeed = max(2000, svr.FanSpeed-100-rand.Intn(200))
		} else {
			svr.FanSpeed += rand.Intn(200) - 100
		}

		powerDrift := (rand.Float64() - 0.5) * 20.0
		svr.PowerUsage = minf(svr.PowerLimit, svr.PowerUsage+powerDrift)
		if svr.PowerUsage < 150.0 {
			svr.PowerUsage = 150.0
		}

		svr.LastUpdated = now
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (s *BMCSimulator) Start(interval time.Duration) chan struct{} {
	done := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.Tick()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return done
}
