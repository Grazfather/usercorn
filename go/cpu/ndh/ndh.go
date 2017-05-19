package ndh

import (
	"binary"

	"github.com/lunixbochs/usercorn/go/models/cpu"
)

type Builder struct{}

var regMap = map[string]int{
	"r0": 0,
	"r1": 1,
	"r2": 2,
	"r3": 3,
	"r4": 4,
	"r5": 5,
	"r6": 6,
	"r7": 7,
	"sp": 8,
	"bp": 9,
	"pc": 10,
}

func (b *Builder) New() (cpu.Cpu, error) {
	ndh = &NdhCpu{}
	// Ugh
	regs := make([]int, len(regMap))
	for i := range regs {
		regs[i] = i
	}

	ndh.Regs = cpu.NewRegs(16, regs)
	ndh.Mem = cpu.NewMem(16, binary.LittleEndian)
	ndh.Hooks = cpu.NewHooks(ndh, ndh.Mem)
	return ndh, _
}

type NdhCpu struct {
	Hooks &cpu.Hooks
	Mem &cpu.Mem
	Regs &cpu.Reg
}

// execution
func (c *NdhCpu) Start(begin, until uint64) error {
	return nil
}
func (c *NdhCpu) Stop() error {
	return nil
}

// cleanup
func (c *NdhCpu) Close() error {
	return nil
}

// leaky abstraction
func (c *NdhCpu) Backend() interface{} {
	return nil
}
