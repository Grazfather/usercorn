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
	AF
	BF
	ZF
)

type Builder struct{}

func (b *Builder) New() (cpu.Cpu, error) {
	// Ugh
	regs := []int{R0, R1, R2, R3, R4, R5, R6, R7, BP, SP, PC, AF, BF, ZF}

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

func (c *NdhCpu) setZf(v bool) {
	if v {
		c.RegWrite(ZF, 1)
	} else {
		c.RegWrite(ZF, 0)
	}
}

// execution
func (c *NdhCpu) Start(begin, until uint64) error {
	var pc uint64
	var jump uint64
	var v uint16
	var v2 uint16
	var data = make([]byte, 2)
	// TODO: Support other exit mechanisms e.g. END
	// TODO: What about jumps before begin?
	// TODO: begin is ignored
	for pc < until {
		// TODO: Check for errors
		pc, _ = c.RegRead(PC)
		jump = 0
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
			v = c.get(instr.args[0]) + c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_SUB:
			v = c.get(instr.args[0]) - c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_MOV:
			v = c.get(instr.args[1])
			c.set(instr.args[0], v)
		case OP_INC:
			v = c.get(instr.args[0]) + 1
			c.set(instr.args[0], v)
		case OP_DEC:
			v = c.get(instr.args[0]) - 1
			c.set(instr.args[0], v)
		case OP_POP:
			sp, _ := c.RegRead(SP)
			data, _ = c.MemRead(uint64(sp), 2)
			v := binary.LittleEndian.Uint16(data)
			sp += 2
			c.RegWrite(SP, sp)
			c.set(instr.args[0], v)
		case OP_PUSH:
			v = c.get(instr.args[0])
			sp, _ := c.RegRead(SP)
			sp -= 2
			c.RegWrite(SP, sp)
			binary.LittleEndian.PutUint16(data, v)
			c.MemWrite(sp, data)
		case OP_JMPS:
			fallthrough
		case OP_JMPL:
			jump = uint64(c.get(instr.args[0]))
		case OP_TEST:
			v = c.get(instr.args[0])
			v2 = c.get(instr.args[1])
			c.setZf(v == 0 && v2 == 0)
		case OP_AND:
			v = c.get(instr.args[0]) & c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_JZ:
			if zf, _ := c.RegRead(ZF); zf == 1 {
				jump = uint64(c.get(instr.args[0]))
			}
		case OP_JNZ:
			if zf, _ := c.RegRead(ZF); zf != 1 {
				jump = uint64(c.get(instr.args[0]))
			}
		case OP_SYSCALL:
			c.OnIntr(0)
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
		case OP_RET:
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
		c.RegWrite(PC, pc+uint64(len(instr.Bytes()))+jump)
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
