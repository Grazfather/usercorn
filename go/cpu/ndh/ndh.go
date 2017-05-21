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
	return ndh, nil
}

type NdhCpu struct {
	*cpu.Regs
	*cpu.Mem
	*cpu.Hooks
	stopRequest bool
	err         error
}

func (c *NdhCpu) set(arg arg, value uint16) error {
	switch a := arg.(type) {
	case *reg:
		c.RegWrite(int(a.num), uint64(value))
	case *indirect:
		addr := c.get(a.arg.(*reg))
		c.WriteUint(uint64(addr), 1, cpu.PROT_WRITE, uint64(value&0xFF))
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
		var val uint64
		addr := c.get(a.arg.(*reg))
		val, c.err = c.ReadUint(uint64(addr), 1, cpu.PROT_READ)
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
	var dis Dis
	var jump uint64
	c.stopRequest = false
	var pc = begin
	c.RegWrite(PC, pc)
	c.OnBlock(pc, 0)

	for pc != until && !c.stopRequest && c.err == nil {
		pc, _ = c.RegRead(PC)
		if jump != 0 {
			c.OnBlock(pc, 0)
		}
		jump = 0
		buf, _ := c.ReadProt(pc, MAX_INST_LEN, cpu.PROT_EXEC)
		instructions, err := dis.Dis(buf, pc)
		if err != nil {
			return err
		}
		instr := instructions[0].(*ins)
		c.OnCode(pc, uint32(len(instr.bytes)))

		if c.stopRequest {
			break
		}

		var a, b arg
		var va, vb uint16
		switch len(instr.args) {
		case 2:
			b = instr.args[1]
			fallthrough
		case 1:
			a = instr.args[0]
		}
		zfcheck := func(v uint16) uint16 {
			if v == 0 {
				c.RegWrite(ZF, 1)
			} else {
				c.RegWrite(ZF, 0)
			}
			return v
		}

		switch instr.op {
		case OP_END:
			return models.ExitStatus(0)
		case OP_ADD:
			c.set(a, zfcheck(c.get(a)+c.get(b)))
		case OP_SUB:
			c.set(a, zfcheck(c.get(a)-c.get(b)))
		case OP_DIV:
			c.set(a, zfcheck(c.get(a)/c.get(b)))
		case OP_MUL:
			c.set(a, zfcheck(c.get(a)*c.get(b)))
		case OP_NOT:
			c.set(a, zfcheck(c.get(a)^^uint16(0)))
		case OP_OR:
			c.set(a, zfcheck(c.get(a)|c.get(b)))
		case OP_AND:
			c.set(a, zfcheck(c.get(a)&c.get(b)))
		case OP_XOR:
			c.set(a, zfcheck(c.get(a)^c.get(b)))
		case OP_MOV:
			c.set(a, c.get(b))
		case OP_XCHG:
			va = c.get(a)
			vb = c.get(b)
			c.set(a, vb)
			c.set(b, va)
		case OP_INC:
			c.set(a, c.get(a)+1)
		case OP_DEC:
			c.set(a, c.get(a)-1)
		case OP_POP:
			var v uint64
			sp, _ := c.RegRead(SP)
			v, c.err = c.ReadUint(sp, 2, cpu.PROT_READ)
			sp += 2
			c.RegWrite(SP, sp)
			c.set(a, uint16(v))
		case OP_PUSH:
			size := 2
			v := c.get(a)
			sp, _ := c.RegRead(SP)
			if _, ok := a.(*a8); ok {
				size = 1
			}
			sp -= uint64(size)
			c.WriteUint(sp, size, cpu.PROT_WRITE, uint64(v))
			c.RegWrite(SP, sp)
		case OP_TEST:
			va = c.get(a)
			vb = c.get(b)
			c.setZf(va == 0 && vb == 0)
		case OP_CMP:
			va = c.get(a)
			vb = c.get(b)
			if va == vb {
				c.setZf(true)
				c.RegWrite(AF, 0)
				c.RegWrite(BF, 0)
			} else {
				c.setZf(false)
				if va > vb {
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
			jump = uint64(c.get(a))
		case OP_JZ:
			if zf, _ := c.RegRead(ZF); zf == 1 {
				jump = uint64(c.get(a))
			}
		case OP_JNZ:
			if zf, _ := c.RegRead(ZF); zf != 1 {
				jump = uint64(c.get(a))
			}
		case OP_JA:
			if af, _ := c.RegRead(AF); af != 1 {
				jump = uint64(c.get(a))
			}
		case OP_JB:
			if bf, _ := c.RegRead(BF); bf != 1 {
				jump = uint64(c.get(a))
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
			va = c.get(a)
			switch a.(type) {
			case *reg:
				c.RegWrite(PC, uint64(va))
				c.OnBlock(uint64(va), 0)
				continue
			case *a16:
				jump = uint64(va)
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
		if jump != 0 {
			pc += uint64(len(instr.Bytes())) + jump
			c.OnBlock(pc, 0)
		} else {
			pc += uint64(len(instr.Bytes()))
		}
		c.RegWrite(PC, pc)
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
