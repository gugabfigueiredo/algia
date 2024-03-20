package cmd

import (
	"fmt"
	"github.com/urfave/cli/v2"
)

const version = "0.0.67"

func DoVersion(cCtx *cli.Context) error {
	fmt.Println(version)
	return nil
}
