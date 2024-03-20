package main

import (
	"fmt"
	"github.com/mattn/algia/adapter/cmd"
	"github.com/mattn/algia/domain"
	"github.com/urfave/cli/v2"
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

func main() {
	app := &cli.App{
		Usage:       "A cli application for nostr",
		Description: "A cli application for nostr",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "a", Usage: "profile name"},
			&cli.StringFlag{Name: "relays", Usage: "relays"},
			&cli.BoolFlag{Name: "V", Usage: "verbose"},
		},
		Commands: []*cli.Command{
			{
				Name:    "timeline",
				Aliases: []string{"tl"},
				Usage:   "show timeline",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Usage: "user"},
					&cli.IntFlag{Name: "n", Value: 30, Usage: "number of items"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
					&cli.BoolFlag{Name: "article", Usage: "show articles"},
				},
				Action: cmd.DoTimeline,
			},
			{
				Name:  "stream",
				Usage: "show stream",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "author"},
					&cli.IntSliceFlag{Name: "kind", Value: cli.NewIntSlice(nostr.KindTextNote)},
					&cli.BoolFlag{Name: "follow"},
					&cli.StringFlag{Name: "pattern"},
					&cli.StringFlag{Name: "reply"},
				},
				Action: cmd.DoStream,
			},
			{
				Name:    "post",
				Aliases: []string{"n"},
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "u", Usage: "users"},
					&cli.BoolFlag{Name: "stdin"},
					&cli.StringFlag{Name: "sensitive"},
					&cli.StringSliceFlag{Name: "emoji"},
					&cli.StringFlag{Name: "geohash"},
					&cli.StringFlag{Name: "article-name"},
					&cli.StringFlag{Name: "article-title"},
					&cli.StringFlag{Name: "article-summary"},
				},
				Usage:     "post new note",
				UsageText: "algia post [note text]",
				HelpName:  "post",
				ArgsUsage: "[note text]",
				Action:    cmd.DoPost,
			},
			{
				Name:    "reply",
				Aliases: []string{"r"},
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "stdin"},
					&cli.StringFlag{Name: "id", Required: true},
					&cli.BoolFlag{Name: "quote"},
					&cli.StringFlag{Name: "sensitive"},
					&cli.StringSliceFlag{Name: "emoji"},
					&cli.StringFlag{Name: "geohash"},
				},
				Usage:     "reply to the note",
				UsageText: "algia reply --id [id] [note text]",
				HelpName:  "reply",
				ArgsUsage: "[note text]",
				Action:    cmd.DoReply,
			},
			{
				Name:    "repost",
				Aliases: []string{"b"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "repost the note",
				UsageText: "algia repost --id [id]",
				HelpName:  "repost",
				Action:    cmd.DoRepost,
			},
			{
				Name:    "unrepost",
				Aliases: []string{"B"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "unrepost the note",
				UsageText: "algia unrepost --id [id]",
				HelpName:  "unrepost",
				Action:    cmd.DoUnrepost,
			},
			{
				Name:    "like",
				Aliases: []string{"l"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
					&cli.StringFlag{Name: "content"},
					&cli.StringFlag{Name: "emoji"},
				},
				Usage:     "like the note",
				UsageText: "algia like --id [id]",
				HelpName:  "like",
				Action:    cmd.DoLike,
			},
			{
				Name:    "unlike",
				Aliases: []string{"L"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "unlike the note",
				UsageText: "algia unlike --id [id]",
				HelpName:  "unlike",
				Action:    cmd.DoUnlike,
			},
			{
				Name:    "delete",
				Aliases: []string{"d"},
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
				},
				Usage:     "delete the note",
				UsageText: "algia delete --id [id]",
				HelpName:  "delete",
				Action:    cmd.DoDelete,
			},
			{
				Name:    "search",
				Aliases: []string{"s"},
				Flags: []cli.Flag{
					&cli.IntFlag{Name: "n", Value: 30, Usage: "number of items"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Usage:     "search notes",
				UsageText: "algia search [words]",
				HelpName:  "search",
				Action:    cmd.DoSearch,
			},
			{
				Name: "broadcast",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Required: true},
					&cli.StringFlag{Name: "relay", Required: false},
				},
				Usage:     "broadcast the note",
				UsageText: "algia broadcast --id [id]",
				HelpName:  "broadcast",
				Action:    cmd.DoBroadcast,
			},
			{
				Name: "dm-list",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				Usage:     "show DM list",
				UsageText: "algia dm-list",
				HelpName:  "dm-list",
				Action:    cmd.DoDMList,
			},
			{
				Name: "dm-timeline",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "DM user", Required: true},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
					&cli.BoolFlag{Name: "extra", Usage: "extra JSON"},
				},
				Usage:     "show DM timeline",
				UsageText: "algia dm-timeline",
				HelpName:  "dm-timeline",
				Action:    cmd.DoDMTimeline,
			},
			{
				Name: "dm-post",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "DM user", Required: true},
					&cli.BoolFlag{Name: "stdin"},
					&cli.StringFlag{Name: "sensitive"},
				},
				Usage:     "post new DM note",
				UsageText: "algia post [note text]",
				HelpName:  "post",
				ArgsUsage: "[note text]",
				Action:    cmd.DoDMPost,
			},
			{
				Name: "bm-list",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				Usage:     "show bookmarks",
				UsageText: "algia bm-list",
				HelpName:  "bm-list",
				Action:    cmd.DoBMList,
			},
			{
				Name:      "bm-post",
				Usage:     "post bookmark",
				UsageText: "algia bm-post [note]",
				HelpName:  "bm-post",
				ArgsUsage: "[note]",
				Action:    cmd.DoBMPost,
			},
			{
				Name: "profile",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "u", Value: "", Usage: "user"},
					&cli.BoolFlag{Name: "json", Usage: "output JSON"},
				},
				Usage:     "show profile",
				UsageText: "algia profile",
				HelpName:  "profile",
				Action:    cmd.DoProfile,
			},
			{
				Name:      "powa",
				Usage:     "post ぽわ〜",
				UsageText: "algia powa",
				HelpName:  "powa",
				Action:    cmd.DoPowa,
			},
			{
				Name:      "puru",
				Usage:     "post ぷる",
				UsageText: "algia puru",
				HelpName:  "puru",
				Action:    cmd.DoPuru,
			},
			{
				Name: "zap",
				Flags: []cli.Flag{
					&cli.Uint64Flag{Name: "amount", Usage: "amount for zap", Value: 1},
					&cli.StringFlag{Name: "comment", Usage: "comment for zap", Value: ""},
				},
				Usage:     "zap something",
				UsageText: "algia zap [note|npub|nevent]",
				HelpName:  "zap",
				Action:    cmd.DoZap,
			},
			{
				Name:      "version",
				Usage:     "show version",
				UsageText: "algia version",
				HelpName:  "version",
				Action:    cmd.DoVersion,
			},
		},
		Before: func(cCtx *cli.Context) error {
			if cCtx.Args().Get(0) == "version" {
				return nil
			}
			profile := cCtx.String("a")
			cfg, err := domain.LoadConfig(profile)
			if err != nil {
				return err
			}
			cCtx.App.Metadata = map[string]any{
				"config": cfg,
			}
			cfg.Verbose = cCtx.Bool("V")
			relays := cCtx.String("relays")
			if strings.TrimSpace(relays) != "" {
				cfg.Relays = make(map[string]domain.Relay)
				for _, relay := range strings.Split(relays, ",") {
					cfg.Relays[relay] = domain.Relay{
						Read:  true,
						Write: true,
					}
				}
				cfg.TempRelay = true
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
