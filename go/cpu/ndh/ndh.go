package ndh

import (
	"encoding/binary"
	"fmt"

	"github.com/lunixbochs/usercorn/go/models"
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
	MAX_INST_LEN = 5
)

type Builder struct{}

func (b *Builder) New() (cpu.Cpu, error) {
	ndh := &NdhCpu{
		Regs: cpu.NewRegs(16, []int{
			R0, R1, R2, R3, R4, R5, R6, R7,
			BP, SP, PC, AF, BF, ZF}),
		Mem: cpu.NewMem(16, binary.LittleEndian),
	}
	ndh.Hooks = cpu.NewHooks(ndh, ndh.Mem)
	ndh.Dis = &Dis{}
	return ndh, nil
}

type NdhCpu struct {
	*cpu.Regs
	*cpu.Mem
	*cpu.Hooks
	Dis         *Dis
	stopRequest bool
	err         error
}

func (c *NdhCpu) set(arg arg, value uint16) error {
	switch a := arg.(type) {
	case *reg:
		c.RegWrite(int(a.num), uint64(value))
	case *indirect:
		var addr uint64
		switch a := a.arg.(type) {
		case *reg:
			addr, _ = c.RegRead(int(a.num))
		default:
			panic("Wtf this indirect has a non-reg arg")
		}
		c.WriteUint(addr, 1, cpu.PROT_WRITE, uint64(value&0xFF))
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
		var addr, val uint64
		switch a := a.arg.(type) {
		case *reg:
			addr, _ = c.RegRead(int(a.num))
		default:
			panic("Wtf this indirect has a non-reg arg")
		}
		val, c.err = c.ReadUint(addr, 1, cpu.PROT_READ)
		v = uint16(val)
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
	c.stopRequest = false
	c.RegWrite(PC, begin)
	c.OnBlock(begin, 0)
	for pc < until && !c.stopRequest && c.err == nil {
		pc, _ = c.RegRead(PC)
		if jump != 0 {
			c.OnBlock(pc, 0)
		}
		jump = 0
		buf, _ := c.ReadProt(pc, MAX_INST_LEN, cpu.PROT_EXEC)
		instructions, err := c.Dis.Dis(buf, pc)
		if err != nil {
			return err
		}
		instr := instructions[0].(*ins)
		c.OnCode(pc, uint32(len(instr.bytes)))
		if c.stopRequest {
			break
		}
		switch instr.op {
		case OP_END:
			return models.ExitStatus(0)
		case OP_ADD:
			v = c.get(instr.args[0]) + c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_SUB:
			v = c.get(instr.args[0]) - c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_DIV:
			v = c.get(instr.args[0]) / c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_MUL:
			v = c.get(instr.args[0]) * c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_NOT:
			v = c.get(instr.args[0]) ^ ^uint16(0)
			c.setZf(v == 0)
		case OP_OR:
			v = c.get(instr.args[0]) | c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_AND:
			v = c.get(instr.args[0]) & c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_XOR:
			v = c.get(instr.args[0]) ^ c.get(instr.args[1])
			c.set(instr.args[0], v)
			c.setZf(v == 0)
		case OP_MOV:
			v = c.get(instr.args[1])
			c.set(instr.args[0], v)
		case OP_XCHG:
			v = c.get(instr.args[0])
			v2 = c.get(instr.args[1])
			c.set(instr.args[0], v2)
			c.set(instr.args[1], v)
		case OP_INC:
			v = c.get(instr.args[0]) + 1
			c.set(instr.args[0], v)
		case OP_DEC:
			v = c.get(instr.args[0]) - 1
			c.set(instr.args[0], v)
		case OP_POP:
			sp, _ := c.RegRead(SP)
			v, _ := c.ReadUint(sp, 2, cpu.PROT_READ)
			sp += 2
			c.RegWrite(SP, sp)
			c.set(instr.args[0], uint16(v))
		case OP_PUSH:
			size := 2
			v = c.get(instr.args[0])
			sp, _ := c.RegRead(SP)
			if _, ok := instr.args[0].(*a8); ok {
				size = 1
			}
			sp -= uint64(size)
			c.WriteUint(sp, size, cpu.PROT_WRITE, uint64(v))
			c.RegWrite(SP, sp)
		case OP_TEST:
			v = c.get(instr.args[0])
			v2 = c.get(instr.args[1])
			c.setZf(v == 0 && v2 == 0)
		case OP_CMP:
			v = c.get(instr.args[0])
			v2 = c.get(instr.args[1])
			if v == v2 {
				c.setZf(true)
				c.RegWrite(AF, 0)
				c.RegWrite(BF, 0)
			} else {
				c.setZf(false)
				if v > v2 {
					c.RegWrite(AF, 1)
					c.RegWrite(BF, 0)
				} else {
					c.RegWrite(AF, 0)
					c.RegWrite(BF, 1)
				}
			}
		case OP_JMPS:
			fallthrough
		case OP_JMPL:
			jump = uint64(c.get(instr.args[0]))
		case OP_JZ:
			if zf, _ := c.RegRead(ZF); zf == 1 {
				jump = uint64(c.get(instr.args[0]))
			}
		case OP_JNZ:
			if zf, _ := c.RegRead(ZF); zf != 1 {
				jump = uint64(c.get(instr.args[0]))
			}
		case OP_JA:
			if af, _ := c.RegRead(AF); af != 1 {
				jump = uint64(c.get(instr.args[0]))
			}
		case OP_JB:
			if bf, _ := c.RegRead(BF); bf != 1 {
				jump = uint64(c.get(instr.args[0]))
			}
		case OP_SYSCALL:
			c.OnIntr(0)
		case OP_CALL:
			// Push RA
			sp, _ := c.RegRead(SP)
			sp -= 2
			c.RegWrite(SP, sp)
			c.WriteUint(sp, 2, cpu.PROT_WRITE, pc+uint64(len(instr.Bytes())))
			// Call is special: If the arg is a register it's an
			// absolute jump, otherwise it's an offset
			v = c.get(instr.args[0])
			switch instr.args[0].(type) {
			case *reg:
				c.RegWrite(PC, uint64(v))
				continue
			case *a16:
				jump = uint64(v)
			}
		case OP_RET:
			// Pop RA
			sp, _ := c.RegRead(SP)
			v, _ := c.ReadUint(uint64(sp), 2, cpu.PROT_READ)
			sp += 2
			c.RegWrite(SP, sp)
			c.RegWrite(PC, v)
			c.OnBlock(v, 0)
			continue
		case OP_NOP:
			// Do nothing
		default:
			fmt.Println("[UNIMPLEMENTED]")
			//return errors.Errorf("Unhandled or illegal instruction! %v\n", instr)
		}
		c.RegWrite(PC, pc+uint64(len(instr.Bytes()))+jump)
	}
	return c.err
}

func (c *NdhCpu) Stop() error {
	c.stopRequest = true
	return nil
}

// cleanup
func (c *NdhCpu) Close() error { return nil }

// leaky abstraction
func (c *NdhCpu) Backend() interface{} {
	return c
}
