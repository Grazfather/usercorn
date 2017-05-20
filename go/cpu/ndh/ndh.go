package ndh

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"

	"github.com/lunixbochs/usercorn/go/models/cpu"
)

const (
	R0 = iota
	R1
	R2
	R3
	R4
	R5
	R6
	R7
	BP
	SP
	PC
)

type Builder struct{}

func (b *Builder) New() (cpu.Cpu, error) {
	// Ugh
	regs := []int{R0, R1, R2, R3, R4, R5, R6, R7, BP, SP, PC}

	ndh := &NdhCpu{Regs: cpu.NewRegs(16, regs), Mem: cpu.NewMem(16, binary.LittleEndian)}
	hooks := cpu.NewHooks(ndh, ndh.Mem)
	ndh.Hooks = hooks
	ndh.Dis = &Dis{}
	return ndh, nil
}

type NdhCpu struct {
	*cpu.Regs
	*cpu.Mem
	*cpu.Hooks
	Dis *Dis
}

func (c *NdhCpu) set(arg arg, value uint16) error {
	switch a := arg.(type) {
	case *reg:
		c.RegWrite(int(a.num), uint64(value))
	case *indirect:
		var addr uint64
		var data []byte
		switch a := a.arg.(type) {
		case *reg:
			addr, _ = c.RegRead(int(a.num))
		default:
			panic("Wtf this indirect has a non-reg arg")
		}
		binary.LittleEndian.PutUint16(data, value)
		c.MemWrite(addr, data)
	}
	return nil
}

func (c *NdhCpu) get(arg arg) uint16 {
	var v uint16
	switch a := arg.(type) {
	case *reg:
		regval, _ := c.RegRead(int(a.num))
		v = uint16(regval)
	case *indirect:
		var addr uint64
		var data []byte
		switch a := a.arg.(type) {
		case *reg:
			addr, _ = c.RegRead(int(a.num))
		default:
			panic("Wtf this indirect has a non-reg arg")
		}
		data, _ = c.MemRead(addr, 2)
		v = binary.LittleEndian.Uint16(data)
	case *a8:
		v = uint16(a.val)
	case *a16:
		v = a.val
	}
	return v
}

// execution
func (c *NdhCpu) Start(begin, until uint64) error {
	var pc uint64
	// TODO: Support other exit mechanisms e.g. END
	// TODO: What about jumps before begin?
	// TODO: begin is ignored
	for pc < until {
		// TODO: Check for errors
		pc, _ = c.RegRead(PC)
		// TODO: How much should I read in? 5bytes is definitely enough for 1
		// Why five? Because it is for sure enough for a whole instruction
		// Ugly
		buf, _ := c.MemRead(pc, 5)
		instructions, err := c.Dis.Dis(buf, pc)
		if err != nil {
			return err
		}
		instr := instructions[0].(*ins)
		switch instr.op {
		case OP_END:
			//return errors.ExitStatus
			return errors.New("END: We're done here and I don't know how to stop")
		case OP_ADD:
			// TODO: ZF
			c.set(instr.args[0], c.get(instr.args[0])+c.get(instr.args[1]))
		case OP_MOV:
			c.set(instr.args[0], c.get(instr.args[1]))
		case OP_INC:
			// TODO: ZF
			c.set(instr.args[0], c.get(instr.args[0])+1)
		case OP_DEC:
			// TODO: ZF
			c.set(instr.args[0], c.get(instr.args[0])-1)
		case OP_AND:
			// TODO: ZF
			fallthrough
		case OP_CALL:
			fallthrough
		case OP_CMP:
			// TODO: AF & BF
			// TODO: ZF
			fallthrough
		case OP_DIV:
			// TODO: ZF
			fallthrough
		case OP_JA:
			fallthrough
		case OP_JB:
			fallthrough
		case OP_JMPL:
			fallthrough
		case OP_JMPS:
			fallthrough
		case OP_JNZ:
			fallthrough
		case OP_JZ:
			fallthrough
		case OP_MUL:
			// TODO: ZF
			fallthrough
		case OP_NOP:
			fallthrough
		case OP_NOT:
			// TODO: ZF
			fallthrough
		case OP_OR:
			// TODO: ZF
			fallthrough
		case OP_POP:
			fallthrough
		case OP_PUSH:
			fallthrough
		case OP_RET:
			fallthrough
		case OP_SUB:
			// TODO: ZF
			fallthrough
		case OP_SYSCALL:
			fallthrough
		case OP_TEST:
			// TODO: ZF
			fallthrough
		case OP_XCHG:
			fallthrough
		case OP_XOR:
			// TODO: ZF
			fallthrough
		default:
			fmt.Println("[UNIMPLEMENTED]")
			//return errors.Errorf("Unhandled or illegal instruction! %v\n", instr)
		}
		c.RegWrite(PC, pc+uint64(len(instr.Bytes())))
	}
	return nil
}

func (c *NdhCpu) Stop() error { return nil }

// cleanup
func (c *NdhCpu) Close() error { return nil }

// leaky abstraction
func (c *NdhCpu) Backend() interface{} {
	return nil
}
