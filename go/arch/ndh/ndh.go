package ndh

import (
	"github.com/lunixbochs/usercorn/go/cpu/ndh"
	co "github.com/lunixbochs/usercorn/go/kernel/common"
	"github.com/lunixbochs/usercorn/go/kernel/linux"
	"github.com/lunixbochs/usercorn/go/kernel/posix"
	"github.com/lunixbochs/usercorn/go/models"
)

type NdhKernel struct {
	*linux.LinuxKernel
}

func NewKernel() *NdhKernel {
	k := &NdhKernel{
		linux.NewKernel(),
	}
	return k
}

func NdhKernels(u models.Usercorn) []interface{} {
	return []interface{}{NewKernel()}
}

var syscallRegs = []int{ndh.R1, ndh.R2, ndh.R3, ndh.R4}
var sysNum = map[int]string{
	0x01: "exit",
	0x02: "open",
	0x03: "read",
	0x04: "write",
	0x05: "close",
	0x06: "setuid",
	0x07: "setgid",
	0x08: "dup2",
	0x09: "send",
	0x0a: "recv",
	0x0b: "socket",
	0x0c: "listen",
	0x0d: "bind",
	0x0e: "accept",
	0x0f: "chdir",
	0x10: "chmod",
	0x11: "lseek",
	0x12: "getpid",
	0x13: "getuid",
	0x14: "pause",
}

func NdhSyscall(u models.Usercorn) {
	num, _ := u.RegRead(ndh.R0)
	name, _ := sysNum[int(num)]
	ret, _ := u.Syscall(int(num), name, co.RegArgs(u, syscallRegs))
	u.RegWrite(ndh.R0, ret)
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
	sp, _ := u.RegRead(u.Arch().SP)
	u.RegWrite(ndh.BP, sp)
	return err
}

// I'll probably use this to dispatch syscalls instead of adding a separate NDH syscall hook
func NdhInterrupt(u models.Usercorn, intno uint32) {
	NdhSyscall(u)
}

func init() {
	Arch.RegisterOS(&models.OS{
		Name:      "ndh",
		Kernels:   NdhKernels,
		Init:      NdhInit,
		Interrupt: NdhInterrupt,
	})
}
