// Command genopenapi prints the OpenAPI document for the current handler
// registration without needing a live database or sandbox: it wires up the
// same huma API as main.go against an unconnected DB handle, since route
// registration only touches type reflection, never a query. Used to refresh
// the frontend's generated API client without the full docker compose stack.
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jmoiron/sqlx"
	"net/http"

	"backend/auth"
	"backend/config"
	"backend/handlers"
	"backend/mailer"
	"backend/sessions"
	"backend/supervisor"
)

func main() {
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://app:app@localhost:5432/app?sslmode=disable")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := sqlx.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	tokens := auth.New(cfg.JWTSecret, cfg.SessionTTL, cfg.ActionTokenTTL)
	google := auth.NewGoogleClient(cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.Google.RedirectURL)
	mail, err := mailer.New(cfg.Email)
	if err != nil {
		log.Fatalf("mailer: %v", err)
	}

	solveSessions := sessions.NewManager(cfg, &supervisor.Supervisor{}, db)
	hs := handlers.New(db, cfg, mail, tokens, google, solveSessions)

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("QuitLARP API", "0.1.0"))
	hs.Register(api)

	out, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		log.Fatalf("marshal openapi: %v", err)
	}
	os.Stdout.Write(out)
}
