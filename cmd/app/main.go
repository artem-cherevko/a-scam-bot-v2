package main

import (
	"context"
	"log"

	"golang.org/x/sync/errgroup"

	"github.com/artem-cherevko/a-scam-bot-v2/internal/api"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/bot"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/config"
	"github.com/artem-cherevko/a-scam-bot-v2/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return bot.StartBot(cfg, ctx)
	})

	g.Go(func() error {
		return api.RunApi(cfg, db)
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
