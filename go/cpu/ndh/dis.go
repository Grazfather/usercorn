package ndh

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strings"

	"github.com/lunixbochs/usercorn/go/models"
)

type ndhReader struct {
	*bytes.Reader
	err  error
	addr uint64
}

func newNdhReader(mem []byte, addr uint64) *ndhReader {
	return &ndhReader{Reader: bytes.NewReader(mem), addr: addr}
}

func (n *ndhReader) tell() int {
	return int(n.Size()) - n.Len()
}

func (n *ndhReader) u8() (b byte) {
	if n.err == nil {
		b, n.err = n.Reader.ReadByte()
	}
	return b
}

func (n *ndhReader) u16() (s uint16) {
	if n.err == nil {
		var tmp [2]byte
		_, n.err = n.Reader.Read(tmp[:])
		s = binary.LittleEndian.Uint16(tmp[:])
	}
	return s
}

type arg interface {
	String() string
}
type ins struct {
	addr  uint64
	op    uint8
	name  string
	args  []arg
	bytes []byte
}

func (i *ins) String() string {
	return i.name + " " + i.OpStr()
}

func (i *ins) Addr() uint64 {
	return i.addr
}

func (i *ins) Bytes() []byte {
	return i.bytes
}

func (i *ins) Mnemonic() string {
	return i.name
}

func (i *ins) OpStr() string {
outer:
	switch i.op {
	case OP_CALL, OP_JA, OP_JB, OP_JMPL, OP_JMPS, OP_JNZ, OP_JZ:
		var addr uint64
		switch a := i.args[0].(type) {
		case *a8:
			addr = (i.addr + uint64(a.val) + uint64(len(i.bytes))) & 0xffff
		case *a16:
			addr = (i.addr + uint64(a.val) + uint64(len(i.bytes))) & 0xffff
		default:
			break outer
		}
		return fmt.Sprintf("%#x", addr)
	}

	args := make([]string, len(i.args))
	for i, v := range i.args {
		args[i] = v.String()
	}
	return strings.Join(args, ", ")
}

type reg struct{ num uint8 }
type a8 struct{ val uint8 }
type a16 struct{ val uint16 }
type indirect struct{ arg }

func (a *reg) String() string {
	switch a.num {
	case PC:
		return "pc"
	case SP:
		return "sp"
	case BP:
		return "bp"
	default:
		return fmt.Sprintf("r%d", a.num)
	}
}

func (a *a8) String() string       { return fmt.Sprintf("%#x", a.val) }
func (a *a16) String() string      { return fmt.Sprintf("%#x", a.val) }
func (i *indirect) String() string { return fmt.Sprintf("[%s]", i.arg) }

func (n *ndhReader) reg() arg          { return &reg{n.u8()} }
func (n *ndhReader) a8() arg           { return &a8{n.u8()} }
func (n *ndhReader) a16() arg          { return &a16{n.u16()} }
func (n *ndhReader) reg_indirect() arg { return &indirect{n.reg()} }

func (n *ndhReader) flag() []arg {
	flag := n.u8()
	switch flag {
	case OP_FLAG_REG_REG:
		return []arg{n.reg(), n.reg()}
	case OP_FLAG_REG_DIRECT08:
		return []arg{n.reg(), n.a8()}
	case OP_FLAG_REG_DIRECT16:
		return []arg{n.reg(), n.a16()}
	case OP_FLAG_REG:
		return []arg{n.reg()}
	case OP_FLAG_DIRECT08:
		return []arg{n.a8()}
	case OP_FLAG_DIRECT16:
		return []arg{n.a16()}
	case OP_FLAG_REGINDIRECT_REG:
		return []arg{n.reg_indirect()}
	case OP_FLAG_REGINDIRECT_DIRECT08:
		return []arg{n.reg_indirect(), n.a8()}
	case OP_FLAG_REGINDIRECT_DIRECT16:
		return []arg{n.reg_indirect(), n.a16()}
	case OP_FLAG_REGINDIRECT_REGINDIRECT:
		return []arg{n.reg_indirect(), n.reg_indirect()}
	case OP_FLAG_REG_REGINDIRECT:
		return []arg{n.reg(), n.reg_indirect()}
	}
	return nil
}

func (n *ndhReader) ins() *ins {
	start := n.tell()
	opcode := n.u8()
	var name string
	var args []arg
	if op, ok := opData[int(opcode)]; ok {
		name = op.name
		switch op.arg {
		case A_NONE:
		case A_1REG:
			args = []arg{n.reg()}
		case A_2REG:
			args = []arg{n.reg(), n.reg()}
		case A_U8:
			args = []arg{n.a8()}
		case A_U16:
			args = []arg{n.a16()}
		case A_FLAG:
			args = n.flag()
		default:
			if n.err == nil {
				n.err = errors.Errorf("unknown op arg type: %d", op.arg)
			}
		}
	} else {
		if n.err == nil {
			n.err = errors.Errorf("unknown opcode: %d", op.arg)
		}
	}
	if n.err == nil {
		len := n.tell() - start
		bytes := make([]byte, len)
		n.ReadAt(bytes, int64(start))
		return &ins{n.addr + uint64(start), opcode, name, args, bytes}
	}
	return nil
}

type Dis struct{}

func (d *Dis) Dis(mem []byte, addr uint64) ([]models.Ins, error) {
	var dis []models.Ins
	var ins *ins
	r := newNdhReader(mem, addr)
	for r.err == nil && (ins == nil || ins.op != OP_END) {
		ins = r.ins()
		if ins != nil {
			dis = append(dis, ins)
		}
	}
	if r.err == io.EOF {
		return dis, nil
	}
	return dis, nil // errors.Wrap(r.err, "disassembly fell short")
}
