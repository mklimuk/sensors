package console

import (
	"fmt"
	"github.com/urfave/cli/v2"
)

func Exit(code int, msg string, args ...interface{}) cli.ExitCoder {
	return cli.Exit(fmt.Sprintf(msg, args...), code)
}
