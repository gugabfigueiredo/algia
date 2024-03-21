package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mattn/algia/internal/domain"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/nostr-sdk"
	"github.com/urfave/cli/v2"
	"os"
	"regexp"
)

func DoStream(cCtx *cli.Context) error {
	kinds := cCtx.IntSlice("kind")
	authors := cCtx.StringSlice("author")
	f := cCtx.Bool("follow")
	pattern := cCtx.String("pattern")
	reply := cCtx.String("reply")

	var re *regexp.Regexp
	if pattern != "" {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return err
		}
	}

	cfg := cCtx.App.Metadata["config"].(*domain.Config)

	relay := cfg.FindRelay(context.Background(), domain.Relay{Read: true})
	if relay == nil {
		return errors.New("cannot connect relays")
	}
	defer relay.Close()

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}
	pub, err := nostr.GetPublicKey(sk)
	if err != nil {
		return err
	}

	// get followers
	var follows []string
	if f {
		followsMap, err := cfg.GetFollows(cCtx.String("a"))
		if err != nil {
			return err
		}
		for k := range followsMap {
			follows = append(follows, k)
		}
	} else {
		for _, author := range authors {
			if pp := sdk.InputToProfile(context.TODO(), author); pp != nil {
				follows = append(follows, pp.PublicKey)
			} else {
				return fmt.Errorf("failed to parse pubkey from '%s'", author)
			}
		}
	}

	since := nostr.Now()
	filter := nostr.Filter{
		Kinds:   kinds,
		Authors: follows,
		Since:   &since,
	}

	sub, err := relay.Subscribe(context.Background(), nostr.Filters{filter})
	if err != nil {
		return err
	}
	for ev := range sub.Events {
		if ev.Kind == nostr.KindTextNote {
			if re != nil && !re.MatchString(ev.Content) {
				continue
			}
			json.NewEncoder(os.Stdout).Encode(ev)
			if reply == "" {
				continue
			}
			var evr nostr.Event
			evr.PubKey = pub
			evr.Content = reply
			evr.Tags = evr.Tags.AppendUnique(nostr.Tag{"e", ev.ID, "", "reply"})
			evr.CreatedAt = nostr.Now()
			evr.Kind = nostr.KindTextNote
			if err := evr.Sign(sk); err != nil {
				return err
			}
			cfg.Do(domain.Relay{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
				relay.Publish(ctx, evr)
				return true
			})
		} else {
			json.NewEncoder(os.Stdout).Encode(ev)
		}
	}
	return nil
}
