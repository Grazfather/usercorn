package ndh

import (
	co "github.com/lunixbochs/usercorn/go/kernel/common"
	"github.com/lunixbochs/usercorn/go/kernel/posix"
	"github.com/lunixbochs/usercorn/go/models"
)

type NdhKernel struct {
	*co.KernelBase
}

func NewKernel() *NdhKernel {
	k := &NdhKernel{
		KernelBase: &co.KernelBase{},
	}
	return k
}

func NdhKernels(u models.Usercorn) []interface{} {
	return []interface{}{NewKernel()}
}

// Map the stack and sets up argc/argv
func NdhInit(u models.Usercorn, args, env []string) error {
	// FIXME: support NX?
	if err := u.MapStack(0, 0x8000, false); err != nil {
		return err
	}
	// push argv strings (ndh has no env)
	addrs, err := posix.PushStrings(u, args...)
	if err != nil {
		return err
	}
	// assumption: I don't think stack has alignment restrictions
	argcArgv := append([]uint64{uint64(len(args))}, addrs...)
	stackBytes, err := posix.PackAddrs(u, argcArgv)
	if err != nil {
		return err
	}
	_, err = u.PushBytes(stackBytes)
	return err
}

// I'll probably use this to dispatch syscalls instead of adding a separate NDH syscall hook
func NdhInterrupt(u models.Usercorn, intno uint32) {
	u.Printf("ndh interrupt: %d\n", intno)
}

func init() {
	Arch.RegisterOS(&models.OS{
		Name:      "ndh",
		Kernels:   NdhKernels,
		Init:      NdhInit,
		Interrupt: NdhInterrupt,
	})
}
