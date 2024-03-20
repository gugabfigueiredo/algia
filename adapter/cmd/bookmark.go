package cmd

import (
	"errors"
	"github.com/mattn/algia/domain"

	"github.com/urfave/cli/v2"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func DoBMList(cCtx *cli.Context) error {
	n := cCtx.Int("n")
	j := cCtx.Bool("json")
	extra := cCtx.Bool("extra")

	cfg := cCtx.App.Metadata["config"].(*domain.Config)

	// get followers
	followsMap, err := cfg.GetFollows(cCtx.String("a"))
	if err != nil {
		return err
	}

	var sk string
	var npub string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	if npub, err = nostr.GetPublicKey(sk); err != nil {
		return err
	}

	// get timeline
	filter := nostr.Filter{
		Kinds:   []int{nostr.KindCategorizedBookmarksList},
		Authors: []string{npub},
		Tags:    nostr.TagMap{"d": []string{"bookmark"}},
		Limit:   n,
	}

	be := []string{}
	evs := cfg.Events(filter)
	for _, ev := range evs {
		for _, tag := range ev.Tags {
			if len(tag) > 1 && tag[0] == "e" {
				be = append(be, tag[1:]...)
			}
		}
	}
	filter = nostr.Filter{
		Kinds: []int{nostr.KindTextNote},
		IDs:   be,
	}
	eevs := cfg.Events(filter)
	cfg.PrintEvents(eevs, followsMap, j, extra)
	return nil
}

func DoBMPost(cCtx *cli.Context) error {
	return errors.New("Not Implemented")
}
