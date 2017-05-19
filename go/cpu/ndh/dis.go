package ndh

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"

	"github.com/lunixbochs/usercorn/go/models"
)

type Dis struct{}

func (d *Dis) Dis(mem []byte, addr uint64) ([]models.Ins, error) {
	var (
		dis     []models.Ins
		opFlags [2]byte
	)
	b := bytes.NewReader(mem)
	for {
		if _, err := b.Read(opFlags[:]); err != nil {
			return dis, errors.Wrap(err, "disassembly fell short")
		}
		fmt.Println(opFlags)
	}
	return dis, nil
}
