package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"backend/auth"
	"backend/config"
	"backend/database"
	"backend/handlers"
	"backend/mailer"
	"backend/middleware"
	"backend/runtime"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL, 30, time.Second)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	if err := database.SeedSuperadmin(context.Background(), db,
		cfg.Superadmin.Username, cfg.Superadmin.Email, cfg.Superadmin.Password); err != nil {
		log.Fatalf("seed superadmin: %v", err)
	}

	tokens := auth.New(cfg.JWTSecret, cfg.SessionTTL, cfg.ActionTokenTTL)
	google := auth.NewGoogleClient(cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.Google.RedirectURL)
	if !google.Enabled() {
		log.Printf("INFO: Google OAuth disabled (GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET not set) — /api/v1/auth/google returns 501")
	}
	attempts := runtime.NewAttemptManager(context.Background(), db)
	hs := handlers.New(db, cfg, mailer.NewMock(cfg.EmailFrom), tokens, google, attempts)

	// The typed, OpenAPI-documented API lives on its own mux...
	apiMux := http.NewServeMux()
	api := humago.New(apiMux, huma.DefaultConfig("QuitLARP API", "0.1.0"))
	hs.Register(api)

	// ...while the browser-redirect OAuth endpoints and the attempt
	// WebSocket are plain http handlers registered directly on the root
	// mux (longer path prefix wins) — the WS upgrade stops being HTTP, so
	// it can't be a huma operation.
	root := http.NewServeMux()
	root.HandleFunc("GET /api/v1/auth/google", hs.GoogleAuthStart)
	root.HandleFunc("GET /api/v1/auth/google/callback", hs.GoogleAuthCallback)
	root.Handle("GET /api/v1/attempts/{id}/ws", middleware.Authenticate(tokens, http.HandlerFunc(hs.AttemptWebSocket)))
	// CookieJar buffers huma responses so session cookies can be attached;
	// Authenticate parses the session token (cookie or Bearer) per request.
	root.Handle("/", middleware.CookieJar(middleware.Authenticate(tokens, apiMux)))

	addr := ":" + cfg.Port
	log.Printf("listening on %s (env=%s)", addr, cfg.Env)
	log.Printf("frontend: %s | openapi: http://localhost%s/openapi.json", cfg.FrontendURL, addr)
	log.Fatal(http.ListenAndServe(addr, middleware.CORS(cfg.CORSOrigins)(root)))
}
