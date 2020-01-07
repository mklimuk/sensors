package console

import (
	"fmt"
	"github.com/urfave/cli"
)

func Exit(code int, msg string, args ...interface{}) *cli.ExitError {
	return cli.NewExitError(fmt.Sprintf(msg, args...), code)
}
