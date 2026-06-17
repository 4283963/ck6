package rpc

import (
	"bmc-rpc-service/bmc"
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
	reply.Servers = s.sim.ListAll()
	return nil
}

func (s *BMCService) GetServerStatus(args GetStatusArgs, reply *GetStatusReply) error {
	status, ok := s.sim.GetStatus(args.ID)
	reply.Status = status
	reply.Ok = ok
	return nil
}

func (s *BMCService) SetPowerLimit(args SetPowerLimitArgs, reply *SetPowerLimitReply) error {
	reply.Ok = s.sim.SetPowerLimit(args.ID, args.Limit)
	return nil
}

func RegisterService(sim *bmc.BMCSimulator) error {
	svc := NewBMCService(sim)
	return rpc.Register(svc)
}

var _ = time.Now
