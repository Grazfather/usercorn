package ndh

import (
	// "github.com/lunixbochs/usercorn/go/cpu"
	"github.com/lunixbochs/usercorn/go/cpu/ndh"
	"github.com/lunixbochs/usercorn/go/models"
)

var Arch = &models.Arch{
	Name: "ndh",
	Bits: 16,

	Cpu: &ndh.Builder{},
	Dis: nil,
	Asm: nil,

	PC: 10,
	SP: 9,
	Regs: map[string]int{
		"r0": 0,
		"r1": 1,
		"r2": 2,
		"r3": 3,
		"r4": 4,
		"r5": 5,
		"r6": 6,
		"r7": 7,
		"bp": 8,
		"sp": 9,
		"pc": 10,
		// TODO: Should we consider the flags as one reg?
		"af": 11,
		"bf": 12,
		"zf": 13,
	},
	DefaultRegs: []string{
		"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7",
	},
}
