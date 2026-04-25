package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"ws_chat/messenger-server/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5"
)

func main() {
	migrationsPath, ok := os.LookupEnv("MIGRATIONS_PATH")
	if !ok {
		panic("migrations-path is required")
	}

	cfg := config.MustLoad()

	conn, err := pgx.Connect(context.Background(), cfg.PostgresURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	m, err := migrate.New(
		"file://"+migrationsPath,
		cfg.PostgresURL,
	)
	if err != nil {
		panic(err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}

		panic(err)
	}

	fmt.Println("migrations applied successfully")
}