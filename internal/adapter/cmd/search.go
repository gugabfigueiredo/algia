package cmd

import (
	"github.com/mattn/algia/internal/domain"
	"github.com/nbd-wtf/go-nostr"
	"github.com/urfave/cli/v2"
	"strings"
)

func DoSearch(cCtx *cli.Context) error {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")

	cfg := cCtx.App.Metadata["config"].(*domain.Config)

	// get followers
	var followsMap map[string]domain.Profile
	var err error
	if j && !extra {
		followsMap = make(map[string]domain.Profile)
	} else {
		followsMap, err = cfg.GetFollows(cCtx.String("a"))
		if err != nil {
			return err
		}
	}

	// get timeline
	filter := nostr.Filter{
		Kinds:  []int{nostr.KindTextNote},
		Search: strings.Join(cCtx.Args().Slice(), " "),
		Limit:  n,
	}

	evs := cfg.Events(filter)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}
