package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mdigger/esl"
)

func main() {
	log.SetFlags(0)

	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("ended", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	if err := initEnv(".env"); err != nil {
		return err
	}

	cfg := struct {
		addr, password string
	}{
		addr:     os.Getenv("ESL_ADDR"),
		password: envDefault("ESL_PASSWORD", "ClueCon"),
	}

	flag.StringVar(&cfg.addr, "addr", cfg.addr, "FreeSWITCH address")
	flag.StringVar(&cfg.password, "password", cfg.password, "FreeSWITCH password")
	flag.Parse()

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer done()

	events := make(chan esl.Event, 1)
	go func() {
		enc := json.NewEncoder(os.Stdout)
		for ev := range events {
			if err := enc.Encode(ev); err != nil {
				slog.Error("failed to encode event", slog.String("err", err.Error()))
				break
			}
		}

		done()
	}()

	client, err := esl.Connect(cfg.addr, cfg.password,
		esl.WithEvents(events),
		esl.WithLog(slog.Default()),
		// esl.WithDumpIn(os.Stdout),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Subscribe(flag.Args()...); err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func envDefault(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return def
}

func initEnv(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open env file: %w", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		// split to key and value
		kv := strings.SplitN(s.Text(), "=", 2)
		if len(kv) != 2 {
			continue
		}

		// skip comments
		if strings.HasPrefix(kv[1], "#") {
			continue
		}

		// skip end of line comments
		v, _, _ := strings.Cut(kv[1], "#")

		// set environment
		if err := os.Setenv(kv[0], strings.TrimSpace(v)); err != nil {
			return fmt.Errorf("failed to set env: %w", err)
		}
	}

	if err := s.Err(); err != nil {
		return fmt.Errorf("failed to parse env file: %w", err)
	}

	return nil
}
