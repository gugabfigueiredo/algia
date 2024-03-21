package cmd

import (
	"github.com/mattn/algia/internal/application"
	"github.com/urfave/cli/v2"
)

func DoPowa(cCtx *cli.Context) error {
	return application.PostMsg(cCtx, "ぽわ〜")
}
