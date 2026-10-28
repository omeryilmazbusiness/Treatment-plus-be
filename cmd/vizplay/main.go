// Command vizplay serves the 3D hamle player without Postgres/Redis.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	vizapp "github.com/treatment-plus/treatment-plus-be/internal/viz/app"
	vizhttp "github.com/treatment-plus/treatment-plus-be/internal/viz/http"
)

func main() {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))
	r.Route("/v1", vizhttp.NewHandler(vizapp.NewService(nil), nil).Register)
	addr := os.Getenv("VIZ_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	log.Printf("viz player http://localhost%s/v1/viz/player", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
