package main

import (
	"context"
	"github.com/example/agnos-test/internal/auth"
	"github.com/example/agnos-test/internal/httpapi"
	"github.com/example/agnos-test/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	db := os.Getenv("DATABASE_URL")
	secret := os.Getenv("JWT_SECRET")
	issuer := os.Getenv("JWT_ISSUER")
	audience := os.Getenv("JWT_AUDIENCE")
	provisioningKey := os.Getenv("PROVISIONING_API_KEY")
	if db == "" || len(secret) < 32 || issuer == "" || audience == "" || len(provisioningKey) < 32 {
		log.Fatal("DATABASE_URL, a 32+ character JWT_SECRET, JWT_ISSUER, JWT_AUDIENCE, and a 32+ character PROVISIONING_API_KEY are required")
	}
	pool, err := pgxpool.New(context.Background(), db)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err = pool.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	p := repository.Postgres{Pool: pool}
	router := httpapi.NewRouter(httpapi.API{Staff: p, Patients: p, Hospitals: p, Tokens: p, Audit: p, JWTConfig: auth.Config{Secret: secret, Issuer: issuer, Audience: audience}, ProvisioningAPIKey: provisioningKey, MaxRequestBodyBytes: 1 << 20})
	server := &http.Server{Addr: ":" + port, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Fatal(server.ListenAndServe())
}
