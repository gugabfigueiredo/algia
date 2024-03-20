package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip19"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config is
type Config struct {
	Relays     map[string]Relay   `json:"relays"`
	Follows    map[string]Profile `json:"follows"`
	PrivateKey string             `json:"privatekey"`
	Updated    time.Time          `json:"updated"`
	Emojis     map[string]string  `json:"emojis"`
	NwcURI     string             `json:"nwc-uri"`
	NwcPub     string             `json:"nwc-pub"`
	Verbose    bool
	TempRelay  bool
	sk         string
}

func ConfigDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		dir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, ".config"), nil
	default:
		return os.UserConfigDir()
	}
}

func LoadConfig(profile string) (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "algia")

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else if profile == "?" {
		names, err := filepath.Glob(filepath.Join(dir, "config-*.json"))
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			name = filepath.Base(name)
			name = strings.TrimLeft(name[6:len(name)-5], "-")
			fmt.Println(name)
		}
		os.Exit(0)
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	os.MkdirAll(filepath.Dir(fp), 0700)

	b, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return nil, err
	}
	if len(cfg.Relays) == 0 {
		cfg.Relays = map[string]Relay{}
		cfg.Relays["wss://relay.nostr.band"] = Relay{
			Read:   true,
			Write:  true,
			Search: true,
		}
	}
	return &cfg, nil
}

// GetFollows is
func (cfg *Config) GetFollows(profile string) (map[string]Profile, error) {
	var mu sync.Mutex
	var pub string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		if pub, err = nostr.GetPublicKey(s.(string)); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	// get followers
	if (cfg.Updated.Add(3*time.Hour).Before(time.Now()) && !cfg.TempRelay) || len(cfg.Follows) == 0 {
		mu.Lock()
		cfg.Follows = map[string]Profile{}
		mu.Unlock()
		m := map[string]struct{}{}

		cfg.Do(Relay{Read: true}, func(ctx context.Context, relay *nostr.Relay) bool {
			evs, err := relay.QuerySync(ctx, nostr.Filter{Kinds: []int{nostr.KindContactList}, Authors: []string{pub}, Limit: 1})
			if err != nil {
				return true
			}
			for _, ev := range evs {
				var rm map[string]Relay
				if cfg.TempRelay == false {
					if err := json.Unmarshal([]byte(ev.Content), &rm); err == nil {
						for k, v1 := range cfg.Relays {
							if v2, ok := rm[k]; ok {
								v2.Search = v1.Search
							}
						}
						cfg.Relays = rm
					}
				}
				for _, tag := range ev.Tags {
					if len(tag) >= 2 && tag[0] == "p" {
						mu.Lock()
						m[tag[1]] = struct{}{}
						mu.Unlock()
					}
				}
			}
			return true
		})
		if cfg.Verbose {
			fmt.Printf("found %d followers\n", len(m))
		}
		if len(m) > 0 {
			follows := []string{}
			for k := range m {
				follows = append(follows, k)
			}

			for i := 0; i < len(follows); i += 500 {
				// Calculate the end index based on the current index and slice length
				end := i + 500
				if end > len(follows) {
					end = len(follows)
				}

				// get follower's descriptions
				cfg.Do(Relay{Read: true}, func(ctx context.Context, relay *nostr.Relay) bool {
					evs, err := relay.QuerySync(ctx, nostr.Filter{
						Kinds:   []int{nostr.KindProfileMetadata},
						Authors: follows[i:end], // Use the updated end index
					})
					if err != nil {
						return true
					}
					for _, ev := range evs {
						var profile Profile
						err := json.Unmarshal([]byte(ev.Content), &profile)
						if err == nil {
							mu.Lock()
							cfg.Follows[ev.PubKey] = profile
							mu.Unlock()
						}
					}
					return true
				})
			}
		}

		cfg.Updated = time.Now()
		if err := cfg.save(profile); err != nil {
			return nil, err
		}
	}
	return cfg.Follows, nil
}

// FindRelay is
func (cfg *Config) FindRelay(ctx context.Context, r Relay) *nostr.Relay {
	for k, v := range cfg.Relays {
		if r.Write && !v.Write {
			continue
		}
		if !cfg.TempRelay && r.Search && !v.Search {
			continue
		}
		if !r.Write && !v.Read {
			continue
		}
		if cfg.Verbose {
			fmt.Printf("trying relay: %s\n", k)
		}
		relay, err := nostr.RelayConnect(ctx, k)
		if err != nil {
			if cfg.Verbose {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			continue
		}
		return relay
	}
	return nil
}

// Do is
func (cfg *Config) Do(r Relay, f func(context.Context, *nostr.Relay) bool) {
	var wg sync.WaitGroup
	ctx := context.Background()
	for k, v := range cfg.Relays {
		if r.Write && !v.Write {
			continue
		}
		if r.Search && !v.Search {
			continue
		}
		if !r.Write && !v.Read {
			continue
		}
		wg.Add(1)
		go func(wg *sync.WaitGroup, k string, v Relay) {
			defer wg.Done()
			relay, err := nostr.RelayConnect(ctx, k)
			if err != nil {
				if cfg.Verbose {
					fmt.Fprintln(os.Stderr, err)
				}
				return
			}
			if !f(ctx, relay) {
				ctx.Done()
			}
			relay.Close()
		}(&wg, k, v)
	}
	wg.Wait()
}

func (cfg *Config) save(profile string) error {
	if cfg.TempRelay {
		return nil
	}
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "algia")

	var fp string
	if profile == "" {
		fp = filepath.Join(dir, "config.json")
	} else {
		fp = filepath.Join(dir, "config-"+profile+".json")
	}
	b, err := json.MarshalIndent(&cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, b, 0644)
}

// Decode is
func (cfg *Config) Decode(ev *nostr.Event) error {
	var sk string
	var pub string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
		if pub, err = nostr.GetPublicKey(s.(string)); err != nil {
			return err
		}
	} else {
		return err
	}
	tag := ev.Tags.GetFirst([]string{"p"})
	sp := pub
	if tag != nil {
		sp = tag.Value()
		if sp != pub {
			if ev.PubKey != pub {
				return errors.New("is not author")
			}
		} else {
			sp = ev.PubKey
		}
	}
	ss, err := nip04.ComputeSharedSecret(sp, sk)
	if err != nil {
		return err
	}
	content, err := nip04.Decrypt(ev.Content, ss)
	if err != nil {
		return err
	}
	ev.Content = content
	return nil
}

// PrintEvents is
func (cfg *Config) PrintEvents(evs []*nostr.Event, followsMap map[string]Profile, j, extra bool) {
	if j {
		if extra {
			var events []Event
			for _, ev := range evs {
				if profile, ok := followsMap[ev.PubKey]; ok {
					events = append(events, Event{
						Event:   ev,
						Profile: profile,
					})
				}
			}
			for _, ev := range events {
				json.NewEncoder(os.Stdout).Encode(ev)
			}
		} else {
			for _, ev := range evs {
				json.NewEncoder(os.Stdout).Encode(ev)
			}
		}
		return
	}

	for _, ev := range evs {
		profile, ok := followsMap[ev.PubKey]
		if ok {
			color.Set(color.FgHiRed)
			fmt.Print(profile.Name)
		} else {
			color.Set(color.FgRed)
			if pk, err := nip19.EncodePublicKey(ev.PubKey); err == nil {
				fmt.Print(pk)
			} else {
				fmt.Print(ev.PubKey)
			}
		}
		color.Set(color.Reset)
		fmt.Print(": ")
		color.Set(color.FgHiBlue)
		if ni, err := nip19.EncodeNote(ev.ID); err == nil {
			fmt.Println(ni)
		} else {
			fmt.Println(ev.ID)
		}
		color.Set(color.Reset)
		fmt.Println(ev.Content)
	}
}

// Events is
func (cfg *Config) Events(filter nostr.Filter) []*nostr.Event {
	var mu sync.Mutex
	found := false
	var m sync.Map
	cfg.Do(Relay{Read: true}, func(ctx context.Context, relay *nostr.Relay) bool {
		mu.Lock()
		if found {
			mu.Unlock()
			return false
		}
		mu.Unlock()
		evs, err := relay.QuerySync(ctx, filter)
		if err != nil {
			return true
		}
		for _, ev := range evs {
			if _, ok := m.Load(ev.ID); !ok {
				if ev.Kind == nostr.KindEncryptedDirectMessage || ev.Kind == nostr.KindCategorizedBookmarksList {
					if err := cfg.Decode(ev); err != nil {
						continue
					}
				}
				m.LoadOrStore(ev.ID, ev)
				if len(filter.IDs) == 1 {
					mu.Lock()
					found = true
					ctx.Done()
					mu.Unlock()
					break
				}
			}
		}
		return true
	})

	keys := []string{}
	m.Range(func(k, v any) bool {
		keys = append(keys, k.(string))
		return true
	})
	sort.Slice(keys, func(i, j int) bool {
		lhs, ok := m.Load(keys[i])
		if !ok {
			return false
		}
		rhs, ok := m.Load(keys[j])
		if !ok {
			return false
		}
		return lhs.(*nostr.Event).CreatedAt.Time().Before(rhs.(*nostr.Event).CreatedAt.Time())
	})
	var evs []*nostr.Event
	for _, key := range keys {
		vv, ok := m.Load(key)
		if !ok {
			continue
		}
		evs = append(evs, vv.(*nostr.Event))
	}
	return evs
}

// ZapInfo is
func (cfg *Config) ZapInfo(pub string) (*Lnurlp, error) {
	relay := cfg.FindRelay(context.Background(), Relay{Read: true})
	if relay == nil {
		return nil, errors.New("cannot connect relays")
	}
	defer relay.Close()

	// get set-metadata
	filter := nostr.Filter{
		Kinds:   []int{nostr.KindProfileMetadata},
		Authors: []string{pub},
		Limit:   1,
	}

	evs := cfg.Events(filter)
	if len(evs) == 0 {
		return nil, errors.New("cannot find user")
	}

	var profile Profile
	err := json.Unmarshal([]byte(evs[0].Content), &profile)
	if err != nil {
		return nil, err
	}

	tok := strings.SplitN(profile.Lud16, "@", 2)
	if err != nil {
		return nil, err
	}
	if len(tok) != 2 {
		return nil, errors.New("receipt address is not valid")
	}

	resp, err := http.Get("https://" + tok[1] + "/.well-known/lnurlp/" + tok[0])
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lp Lnurlp
	err = json.NewDecoder(resp.Body).Decode(&lp)
	if err != nil {
		return nil, err
	}
	return &lp, nil
}
