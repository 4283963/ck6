package bmc

import (
	"math/rand"
	"sync"
	"time"
)

const (
	FanMinSpeed    = 2000
	FanMaxSpeed    = 8000
	SilentFanRatio = 0.30
	SilentTempThreshold = 50.0
)

type ServerStatus struct {
	ID              string    `json:"id"`
	FanSpeed        int       `json:"fan_speed"`
	CPUTemp         float64   `json:"cpu_temp"`
	PowerUsage      float64   `json:"power_usage"`
	PowerLimit      float64   `json:"power_limit"`
	FanSilentLimited bool    `json:"fan_silent_limited"`
	LastUpdated     time.Time `json:"last_updated"`
}

type BMCSimulator struct {
	servers    map[string]*ServerStatus
	mu         sync.RWMutex
	silentMode bool
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

func (s *BMCSimulator) SetSilentMode(enabled bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.silentMode = enabled

	if s.servers == nil {
		return
	}

	silentMaxSpeed := int(float64(FanMaxSpeed) * SilentFanRatio)

	for _, svr := range s.servers {
		if svr == nil {
			continue
		}
		if enabled {
			if svr.CPUTemp < SilentTempThreshold {
				if svr.FanSpeed > silentMaxSpeed {
					svr.FanSpeed = silentMaxSpeed
				}
				svr.FanSilentLimited = true
			}
		} else {
			svr.FanSilentLimited = false
		}
	}
}

func (s *BMCSimulator) GetSilentMode() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.silentMode
}

func (s *BMCSimulator) GetSilentModeStats() (enabled bool, total int, limited int) {
	if s == nil || s.servers == nil {
		return false, 0, 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	enabled = s.silentMode
	total = len(s.servers)
	limited = 0

	for _, svr := range s.servers {
		if svr != nil && svr.FanSilentLimited {
			limited++
		}
	}
	return
}

func (s *BMCSimulator) Tick() {
	if s == nil || s.servers == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	silentMaxSpeed := int(float64(FanMaxSpeed) * SilentFanRatio)

	for _, svr := range s.servers {
		if svr == nil {
			continue
		}
		if svr.PowerLimit <= 0 {
			svr.PowerLimit = 500.0
		}

		tempDrift := (rand.Float64() - 0.5) * 3.0
		targetTemp := 35.0 + (svr.PowerUsage / 800.0) * 50.0
		svr.CPUTemp = svr.CPUTemp*0.92 + targetTemp*0.08 + tempDrift

		if s.silentMode {
			if svr.CPUTemp >= SilentTempThreshold {
				svr.FanSilentLimited = false
				if svr.CPUTemp > 70.0 {
					svr.FanSpeed = min(FanMaxSpeed, svr.FanSpeed+200+rand.Intn(300))
				} else {
					svr.FanSpeed += rand.Intn(200) - 50
				}
			} else {
				svr.FanSilentLimited = true
				targetFan := silentMaxSpeed - rand.Intn(200)
				if svr.FanSpeed > targetFan {
					svr.FanSpeed = max(FanMinSpeed, svr.FanSpeed-50-rand.Intn(100))
				} else if svr.FanSpeed < targetFan {
					svr.FanSpeed = min(silentMaxSpeed, svr.FanSpeed+20+rand.Intn(50))
				}
				if svr.FanSpeed > silentMaxSpeed {
					svr.FanSpeed = silentMaxSpeed
				}
			}
		} else {
			svr.FanSilentLimited = false
			if svr.CPUTemp > 70.0 {
				svr.FanSpeed = min(FanMaxSpeed, svr.FanSpeed+200+rand.Intn(300))
			} else if svr.CPUTemp < 50.0 {
				svr.FanSpeed = max(FanMinSpeed, svr.FanSpeed-100-rand.Intn(200))
			} else {
				svr.FanSpeed += rand.Intn(200) - 100
			}
		}

		svr.FanSpeed = max(FanMinSpeed, min(FanMaxSpeed, svr.FanSpeed))

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
