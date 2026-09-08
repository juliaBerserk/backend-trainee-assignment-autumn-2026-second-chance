package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/avito-kitchen/avito_kitchen/internal/handler/api"
	"github.com/avito-kitchen/avito_kitchen/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := env("DATABASE_URL", "postgres://kitchen:kitchen@localhost:5432/avito_kitchen?sslmode=disable")
	addr := env("HTTP_ADDR", ":8080")

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := storage.New(db)
	if err := store.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, api.New(store).Routes()))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
