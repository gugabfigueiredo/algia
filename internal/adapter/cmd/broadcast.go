package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/mattn/algia/internal/domain"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/nostr-sdk"
	"github.com/urfave/cli/v2"
	"os"
	"sync"
	"sync/atomic"
)

func DoBroadcast(cCtx *cli.Context) error {
	id := cCtx.String("id")
	from := cCtx.String("relay")

	var filter nostr.Filter

	if evp := sdk.InputToEventPointer(id); evp == nil {
		epp := sdk.InputToProfile(context.Background(), id)
		if epp == nil {
			return fmt.Errorf("failed to parse note/npub from '%s'", id)
		}
		filter = nostr.Filter{
			Kinds:   []int{nostr.KindProfileMetadata},
			Authors: []string{epp.PublicKey},
		}
	} else {
		filter = nostr.Filter{
			IDs: []string{evp.ID},
		}
	}

	cfg := cCtx.App.Metadata["config"].(*domain.Config)

	var ev *nostr.Event
	var mu sync.Mutex

	if from != "" {
		ctx := context.Background()
		relay, err := nostr.RelayConnect(ctx, from)
		if err != nil {
			return err
		}
		defer relay.Close()
		evs, err := relay.QuerySync(ctx, filter)
		if err != nil {
			return err
		}
		if len(evs) > 0 {
			ev = evs[0]
		}
	} else {
		cfg.Do(domain.Relay{Read: true}, func(ctx context.Context, relay *nostr.Relay) bool {
			if relay.URL == from {
				return true
			}
			evs, err := relay.QuerySync(ctx, filter)
			if err != nil {
				return true
			}
			if len(evs) > 0 {
				mu.Lock()
				ev = evs[0]
				mu.Unlock()
			}
			return false
		})
	}

	if ev == nil {
		return fmt.Errorf("failed to get event '%s'", id)
	}

	var success atomic.Int64
	cfg.Do(domain.Relay{Write: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		err := relay.Publish(ctx, *ev)
		if err != nil {
			fmt.Fprintln(os.Stderr, relay.URL, err)
		} else {
			success.Add(1)
		}
		return true
	})
	if success.Load() == 0 {
		return errors.New("cannot broadcast")
	}
	return nil
}
