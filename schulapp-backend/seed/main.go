package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	email     string
	password  string
	firstName string
	lastName  string
	role      string
}

var users = []seedUser{
	{"admin@schule.de", "admin123", "Max", "Admin", "admin"},
	{"lehrer@schule.de", "lehrer123", "Anna", "Schmidt", "teacher"},
	{"schueler@schule.de", "schueler123", "Tim", "Müller", "student"},
	{"schueler2@schule.de", "schueler123", "Laura", "Weber", "student"},
}

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/schulapp?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("connect:", err)
	}
	defer pool.Close()

	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), 12)
		if err != nil {
			log.Fatal("hash:", err)
		}
		_, err = pool.Exec(context.Background(),
			`INSERT INTO users (email, password_hash, first_name, last_name, role)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (email) DO NOTHING`,
			u.email, string(hash), u.firstName, u.lastName, u.role)
		if err != nil {
			log.Fatal("insert:", err)
		}
		fmt.Printf("seeded: %s (%s)\n", u.email, u.role)
	}
	fmt.Println("seed complete")
}
