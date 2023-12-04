package main

import (
	"context"
	"net/http"
	"os"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/nbd-wtf/go-nostr/nip13"
	"github.com/rs/zerolog"
)

var (
	log   = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	relay = khatru.NewRelay()
)

var supportedKinds = []uint16{
	1,
}

func main() {
	// load db
	db := badger.BadgerBackend{Path: "./db"}
	if err := db.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
		return
	}
	log.Debug().Str("path", "/db").Msg("initialized database")

	// init relay
	// set up some basic properties (will be returned on the NIP-11 endpoint)
	relay.Info.Name = "my relay"
	relay.Info.PubKey = "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	relay.Info.Description = "this is my custom relay"
	relay.Info.Icon = "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fliquipedia.net%2Fcommons%2Fimages%2F3%2F35%2FSCProbe.jpg&f=1&nofb=1&ipt=0cbbfef25bce41da63d910e86c3c343e6c3b9d63194ca9755351bb7c2efa3359&ipo=images"
	relay.Info.Limitation = &nip11.RelayLimitationDocument{
		RestrictedWrites: true,
	}

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.RejectEvent = append(relay.RejectEvent,
		func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
			if nip13.Difficulty(event.ID) < 21 {
				return true, "Below difficulty requirement"
			}
			return false, ""
		},
		policies.PreventLargeTags(100),
		policies.RestrictToSpecifiedKinds(supportedKinds...),
	)

	log.Info().Msg("running on http://0.0.0.0:3334")
	if err := http.ListenAndServe(":3334", relay); err != nil {
		log.Fatal().Err(err).Msg("failed to serve")
	}
}
