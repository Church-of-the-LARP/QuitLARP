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
	"backend/sessions"
	"backend/supervisor"
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

	mail, err := mailer.New(cfg.Email)
	if err != nil {
		log.Fatalf("mailer: %v", err)
	}
	log.Printf("email: provider=%s from=%s", cfg.Email.Provider, cfg.Email.From)

	// The sandboxed assessment environments run inside the dind daemon
	// (gVisor runsc runtime). When it is unreachable the app still serves
	// everything else; starting a solving session then fails with a clear
	// error instead.
	sandbox := &supervisor.Supervisor{}
	if err := sandbox.Init(); err != nil {
		log.Printf("WARNING: assessment sandboxes are unavailable: %v", err)
	} else if err := sandbox.Health(context.Background()); err != nil {
		log.Printf("WARNING: the assessment sandbox is not reachable yet: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	solveSessions := sessions.NewManager(cfg, sandbox, db)
	solveSessions.CleanupOrphans(ctx)
	go solveSessions.Janitor(ctx)

	hs := handlers.New(db, cfg, mail, tokens, google, solveSessions)

	// The typed, OpenAPI-documented API lives on its own mux...
	apiMux := http.NewServeMux()
	api := humago.New(apiMux, huma.DefaultConfig("QuitLARP API", "0.1.0"))
	hs.Register(api)

	// ...while the browser-redirect OAuth endpoints are plain http handlers
	// registered directly on the root mux (longer path prefix wins).
	root := http.NewServeMux()
	root.HandleFunc("GET /api/v1/auth/google", hs.GoogleAuthStart)
	root.HandleFunc("GET /api/v1/auth/google/callback", hs.GoogleAuthCallback)
	// The local git server (clone/fetch/push over smart HTTP) is raw too: it
	// streams a binary protocol and accepts its own Basic-auth credentials.
	root.Handle("/git/", hs.GitHandler())
	// Callback posted by the git update hook during a push: an internal
	// endpoint, not part of the public API.
	root.Handle("POST /internal/push-validation", hs.PushValidationHandler())
	// CookieJar buffers huma responses so session cookies can be attached;
	// Authenticate parses the session token (cookie or Bearer) per request.
	root.Handle("/", middleware.CookieJar(middleware.Authenticate(tokens, apiMux)))

	addr := ":" + cfg.Port
	log.Printf("listening on %s (env=%s)", addr, cfg.Env)
	log.Printf("frontend: %s | openapi: http://localhost%s/openapi.json", cfg.FrontendURL, addr)
	log.Fatal(http.ListenAndServe(addr, middleware.CORS(cfg.CORSOrigins)(root)))
}
