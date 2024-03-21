package cmd

import (
	"context"
	"fmt"
	"github.com/mattn/algia/internal/domain"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/nostr-sdk"
	"github.com/urfave/cli/v2"
)

func DoTimeline(cCtx *cli.Context) error {
	u := cCtx.String("u")
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")
	article := cCtx.Bool("article")

	cfg := cCtx.App.Metadata["config"].(*domain.Config)

	// get followers
	followsMap, err := cfg.GetFollows(cCtx.String("a"))
	if err != nil {
		return err
	}
	var follows []string
	if u == "" {
		for k := range followsMap {
			follows = append(follows, k)
		}
	} else {
		if pp := sdk.InputToProfile(context.TODO(), u); pp != nil {
			u = pp.PublicKey
		} else {
			return fmt.Errorf("failed to parse pubkey from '%s'", u)
		}
		follows = []string{u}
	}

	kind := nostr.KindTextNote
	if article {
		kind = nostr.KindArticle
	}
	// get timeline
	filter := nostr.Filter{
		Kinds:   []int{kind},
		Authors: follows,
		Limit:   n,
	}

	evs := cfg.Events(filter)
	cfg.PrintEvents(evs, followsMap, j, extra)
	return nil
}
