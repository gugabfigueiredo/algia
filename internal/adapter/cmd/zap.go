package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mattn/algia/internal/domain"
	"github.com/mdp/qrterminal/v3"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"os"
)

func pay(cfg *domain.Config, invoice string) error {
	uri, err := url.Parse(cfg.NwcURI)
	if err != nil {
		return err
	}
	wallet := uri.Host
	host := uri.Query().Get("relay")
	secret := uri.Query().Get("secret")
	pub, err := nostr.GetPublicKey(secret)
	if err != nil {
		return err
	}

	relay, err := nostr.RelayConnect(context.Background(), host)
	if err != nil {
		return err
	}
	defer relay.Close()

	ss, err := nip04.ComputeSharedSecret(wallet, secret)
	if err != nil {
		return err
	}
	var req domain.PayRequest
	req.Method = "pay_invoice"
	req.Params.Invoice = invoice
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	content, err := nip04.Encrypt(string(b), ss)
	if err != nil {
		return err
	}

	ev := nostr.Event{
		PubKey:    pub,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindNWCWalletRequest,
		Tags:      nostr.Tags{nostr.Tag{"p", wallet}},
		Content:   content,
	}
	err = ev.Sign(secret)
	if err != nil {
		return err
	}

	filters := []nostr.Filter{{
		Tags: nostr.TagMap{
			"p": []string{pub},
			"e": []string{ev.ID},
		},
		Kinds: []int{nostr.KindNWCWalletInfo, nostr.KindNWCWalletResponse, nostr.KindNWCWalletRequest},
		Limit: 1,
	}}
	sub, err := relay.Subscribe(context.Background(), filters)
	if err != nil {
		return err
	}

	err = relay.Publish(context.Background(), ev)
	if err != nil {
		return err
	}

	er := <-sub.Events
	content, err = nip04.Decrypt(er.Content, ss)
	if err != nil {
		return err
	}
	var resp domain.PayResponse
	err = json.Unmarshal([]byte(content), &resp)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return fmt.Errorf(resp.Err.Message)
	}
	json.NewEncoder(os.Stdout).Encode(resp)
	return nil
}

func DoZap(cCtx *cli.Context) error {
	amount := cCtx.Uint64("amount")
	comment := cCtx.String("comment")
	if cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}

	if cCtx.Args().Len() == 0 {
		return cli.ShowSubcommandHelp(cCtx)
	}

	cfg := cCtx.App.Metadata["config"].(*domain.Config)

	var sk string
	if _, s, err := nip19.Decode(cfg.PrivateKey); err == nil {
		sk = s.(string)
	} else {
		return err
	}

	receipt := ""
	zr := nostr.Event{}
	zr.Tags = nostr.Tags{}

	if pub, err := nostr.GetPublicKey(sk); err == nil {
		if _, err := nip19.EncodePublicKey(pub); err != nil {
			return err
		}
		zr.PubKey = pub
	} else {
		return err
	}

	zr.Tags = zr.Tags.AppendUnique(nostr.Tag{"amount", fmt.Sprint(amount * 1000)})
	relays := nostr.Tag{"relays"}
	for k, v := range cfg.Relays {
		if v.Write {
			relays = append(relays, k)
		}
	}
	zr.Tags = zr.Tags.AppendUnique(relays)
	if prefix, s, err := nip19.Decode(cCtx.Args().First()); err == nil {
		switch prefix {
		case "nevent":
			receipt = s.(nostr.EventPointer).Author
			zr.Tags = zr.Tags.AppendUnique(nostr.Tag{"p", receipt})
			zr.Tags = zr.Tags.AppendUnique(nostr.Tag{"e", s.(nostr.EventPointer).ID})
		case "note":
			evs := cfg.Events(nostr.Filter{IDs: []string{s.(string)}})
			if len(evs) != 0 {
				receipt = evs[0].PubKey
				zr.Tags = zr.Tags.AppendUnique(nostr.Tag{"p", receipt})
			}
			zr.Tags = zr.Tags.AppendUnique(nostr.Tag{"e", s.(string)})
		case "npub":
			receipt = s.(string)
			zr.Tags = zr.Tags.AppendUnique(nostr.Tag{"p", receipt})
		default:
			return errors.New("invalid argument")
		}
	}

	zr.Kind = nostr.KindZapRequest // 9734
	zr.CreatedAt = nostr.Now()
	zr.Content = comment
	if err := zr.Sign(sk); err != nil {
		return err
	}
	b, err := zr.MarshalJSON()
	if err != nil {
		return err
	}

	zi, err := cfg.ZapInfo(receipt)
	if err != nil {
		return err
	}
	u, err := url.Parse(zi.Callback)
	if err != nil {
		return err
	}
	param := url.Values{}
	param.Set("amount", fmt.Sprint(amount*1000))
	param.Set("nostr", string(b))
	u.RawQuery = param.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var iv domain.Invoice
	err = json.NewDecoder(resp.Body).Decode(&iv)
	if err != nil {
		return err
	}

	if cfg.NwcURI == "" {
		config := qrterminal.Config{
			HalfBlocks: false,
			Level:      qrterminal.L,
			Writer:     os.Stdout,
			WhiteChar:  qrterminal.WHITE,
			BlackChar:  qrterminal.BLACK,
			QuietZone:  2,
			WithSixel:  true,
		}
		fmt.Println("lightning:" + iv.PR)
		qrterminal.GenerateWithConfig("lightning:"+iv.PR, config)
	} else {
		pay(cCtx.App.Metadata["config"].(*domain.Config), iv.PR)
	}
	return nil
}
