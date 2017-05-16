package cmd

import (
	"fmt"
	"github.com/lunixbochs/argjoy"
	"github.com/lunixbochs/go-shellwords"
	"github.com/pkg/errors"
	"io"
	"reflect"

	"github.com/lunixbochs/usercorn/go/models"
)

type Command struct {
	Alias []string
	Name  string
	Desc  string
	Run   interface{}
}

var Commands = make(map[string]*Command)

var HelpCmd = registerCommand(&Command{
	Name: "help",
	Desc: "Print basic help",
	Run: func(c *Context, name string) error {
		if name != "all" {
			if command, ok := Commands[name]; ok {
				// Print longer help if it exists
				printCmd(c, command)
				return nil
			} else {
				c.Printf("Could not find command '%s'\n", name)
				return errors.New("X")
			}
		} else {
			for _, command := range Commands {
				printCmd(c, command)
			}
		}
		return nil
	},
})

func printCmd(c *Context, command *Command) {
	c.Printf("%8s : %s\n", command.Name, command.Desc)
}

func registerCommand(c *Command) *Command {
	fn := reflect.ValueOf(c.Run)
	if !fn.IsValid() || fn.Kind() != reflect.Func {
		panic(fmt.Sprintf("Command.Run must be a func: got (%T) %#v\n", c.Run, c.Run))
	}
	Commands[c.Name] = c
	for _, alias := range c.Alias {
		Commands[alias] = c
	}
	return c
}

type Context struct {
	io.ReadWriter
	U models.Usercorn
}

func (c *Context) Printf(format string, a ...interface{}) (int, error) {
	n, err := fmt.Fprintf(c, format, a...)
	return n, errors.Wrap(err, "fmt.Printf() failed")
}

var aj *argjoy.Argjoy

func strCodec(arg interface{}, vals []interface{}) error {
	if a, ok := vals[0].(string); ok {
		if v, ok := arg.(*string); ok {
			*v = a
			return nil
		}
	}
	return argjoy.NoMatch
}

func InitCmds() error {
	aj = argjoy.NewArgjoy()
	aj.Register(strCodec)
	aj.Register(argjoy.RadStrToInt)

	return nil
}

func Dispatch(c *Context, line string) error {
	args, err := shellwords.Parse(line)
	if err != nil {
		c.Printf("parse error: %v\n", err)
		return nil
	}
	if len(args) == 0 {
		return nil
	}
	name, args := args[0], args[1:]
	if cmd, ok := Commands[name]; ok {
		out, err := aj.Call(cmd.Run, c, args)
		if err != nil {
			c.Printf("error: %v\n", err)
		}
		if len(out) > 0 {
			if err, ok := out[0].(error); ok {
				c.Printf("error: %v\n", err)
			}
		}
	} else {
		c.Printf("command not found.\n")
	}
	return nil
}
