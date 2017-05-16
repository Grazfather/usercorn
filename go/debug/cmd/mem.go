package cmd

import (
	"errors"
	"strings"

	"github.com/lunixbochs/usercorn/go/models"
)

var MapsCmd = register(&Command{
	Name: "maps",
	Desc: "Display memory mappings.",
	// TODO: once we have command overloading, merge this with mem command
	Run: func(c *Context) error {
		for _, m := range c.U.Mappings() {
			c.Printf("  %v\n", m.String())
		}
		return nil
	},
})

var MemCmd = register(&Command{
	Name: "mem",
	Desc: "Dump memory.",
	// TODO: need overloading so we can keep arg safety
	// at that point optional args might as well be an overloaded form
	Run: func(c *Context, addr, size uint64) error {
		mem, err := c.U.MemRead(addr, size)
		if err != nil {
			return err
		}
		for _, line := range models.HexDump(addr, mem, int(c.U.Bits())) {
			c.Printf("  %s\n", line)
		}
		return nil
	},
})

var WriteCmd = register(&Command{
	Name: "e",
	Desc: "Write value to memory.",
	// TODO: Need the right codec to allow optional args
	Run: func(c *Context, addr, value, optSize uint64) error {
		s := c.U.StrucAt(addr)
		var err error
		switch optSize {
		case 8:
			err = s.Pack(uint64(value))
		case 4:
			err = s.Pack(uint32(value))
		case 2:
			err = s.Pack(uint16(value))
		case 1:
			err = s.Pack(uint8(value))
		default:
			err = errors.New("Need explicit size")
		}
		if err != nil {
			return err
		}
		return nil
	},
})

var WriteStrCmd = register(&Command{
	Name: "es",
	Desc: "Write string to memory.",
	Run: func(c *Context, addr uint64, str ...string) error {
		sb := []byte(strings.Join(str, " "))
		err := c.U.MemWrite(addr, sb)
		if err != nil {
			return err
		}
		return nil
	},
})
