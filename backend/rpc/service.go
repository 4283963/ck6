package rpc

import (
	"bmc-rpc-service/bmc"
	"fmt"
	"net/rpc"
	"time"
)

type BMCService struct {
	sim *bmc.BMCSimulator
}

type ServerListArgs struct{}

type ServerListReply struct {
	Servers []bmc.ServerStatus `json:"servers"`
}

type GetStatusArgs struct {
	ID string `json:"id"`
}

type GetStatusReply struct {
	Status bmc.ServerStatus `json:"status"`
	Ok     bool             `json:"ok"`
}

type SetPowerLimitArgs struct {
	ID    string  `json:"id"`
	Limit float64 `json:"limit"`
}

type SetPowerLimitReply struct {
	Ok bool `json:"ok"`
}

func NewBMCService(sim *bmc.BMCSimulator) *BMCService {
	return &BMCService{sim: sim}
}

func (s *BMCService) ListServers(args ServerListArgs, reply *ServerListReply) error {
	if s == nil || s.sim == nil || reply == nil {
		return nil
	}
	reply.Servers = s.sim.ListAll()
	return nil
}

func (s *BMCService) GetServerStatus(args GetStatusArgs, reply *GetStatusReply) error {
	if s == nil || s.sim == nil || reply == nil {
		return nil
	}
	status, ok := s.sim.GetStatus(args.ID)
	reply.Status = status
	reply.Ok = ok
	return nil
}

func (s *BMCService) SetPowerLimit(args SetPowerLimitArgs, reply *SetPowerLimitReply) error {
	if s == nil || s.sim == nil || reply == nil {
		return nil
	}
	reply.Ok = s.sim.SetPowerLimit(args.ID, args.Limit)
	return nil
}

func RegisterService(sim *bmc.BMCSimulator) error {
	if sim == nil {
		return fmt.Errorf("simulator cannot be nil")
	}
	svc := NewBMCService(sim)
	return rpc.Register(svc)
}

var _ = fmt.Sprintf

var _ = time.Now
