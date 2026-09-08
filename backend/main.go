package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

type User struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type HealthOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

type UserListOutput struct {
	Body struct {
		Users []User `json:"users"`
	}
}

type UserCreateInput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type UserCreateOutput struct {
	Body struct {
		User User `json:"user"`
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func connectDB(dsn string) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	for range 30 {
		db, err = sqlx.Connect("pgx", dsn)
		if err == nil {
			return db, nil
		}
		log.Printf("database not ready: %v", err)
		time.Sleep(time.Second)
	}
	return nil, err
}

func register(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "getHealth",
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		Summary:     "Check service health",
	}, func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		if err := db.PingContext(ctx); err != nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "database unavailable")
		}
		resp := &HealthOutput{}
		resp.Body.Status = "ok"
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listUsers",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List users",
	}, func(ctx context.Context, _ *struct{}) (*UserListOutput, error) {
		users := []User{}
		if err := db.SelectContext(ctx, &users, "SELECT id, name FROM users ORDER BY id"); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "could not list users")
		}
		resp := &UserListOutput{}
		resp.Body.Users = users
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createUser",
		Method:      http.MethodPost,
		Path:        "/api/v1/users",
		Summary:     "Create a user",
	}, func(ctx context.Context, input *UserCreateInput) (*UserCreateOutput, error) {
		name := strings.TrimSpace(input.Body.Name)
		if name == "" {
			return nil, huma.NewError(http.StatusBadRequest, "name is required")
		}
		var id int64
		if err := db.GetContext(ctx, &id, "INSERT INTO users (name) VALUES ($1) RETURNING id", name); err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "could not create user")
		}
		resp := &UserCreateOutput{}
		resp.Body.User = User{ID: id, Name: name}
		return resp, nil
	})
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations)
	goose.SetDialect("postgres")
	if err := goose.Up(db.DB, "migrations"); err != nil {
		log.Fatalf("could not run migrations: %v", err)
	}

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("CodingTest API", "0.1.0"))
	register(api, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}
