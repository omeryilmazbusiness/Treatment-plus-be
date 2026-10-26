package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/logger"
)

func main() {
	dir := flag.String("dir", "db/migrations", "migrations directory")
	command := flag.String("command", "up", "up|down|version")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(cfg.App.LogLevel, cfg.App.LogFormat)

	source := "file://" + *dir
	m, err := migrate.New(source, cfg.Database.URL)
	if err != nil {
		log.Error("migrate open", "error", err)
		os.Exit(1)
	}
	defer func() { _, _ = m.Close() }()

	switch *command {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "version":
		var v uint
		var dirty bool
		v, dirty, err = m.Version()
		if err == nil {
			log.Info("migrate version", "version", v, "dirty", dirty)
			return
		}
	default:
		err = fmt.Errorf("unknown command %q", *command)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	log.Info("migrate ok", "command", *command)
}
