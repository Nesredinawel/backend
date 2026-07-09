package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run tools/genadmintoken hash <password>")
		fmt.Println("  go run tools/genadmintoken token <jwt_secret> <user_id> [role]")
		return
	}

	switch os.Args[1] {
	case "hash":
		if len(os.Args) < 3 {
			log.Fatal("Usage: go run tools/genadmintoken hash <password>")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[2]), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(hash))

	case "token":
		if len(os.Args) < 4 {
			log.Fatal("Usage: go run tools/genadmintoken token <jwt_secret> <user_id> [role]")
		}
		secret := os.Args[2]
		userID := os.Args[3]
		role := "admin"
		if len(os.Args) > 4 {
			role = os.Args[4]
		}

		now := time.Now().UTC()
		accessClaims := jwt.MapClaims{
			"sub": userID,
			"iss": "auth-service",
			"aud": "hasura-backend",
			"iat": now.Unix(),
			"exp": now.Add(24 * time.Hour).Unix(),
			"jti": uuid.New().String(),
			"https://hasura.io/jwt/claims": map[string]interface{}{
				"x-hasura-default-role":  role,
				"x-hasura-allowed-roles": []string{"user", "admin"},
				"x-hasura-user-id":       userID,
			},
		}

		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("=== Admin JWT (24h expiry) ===")
		fmt.Println(token)
		fmt.Println()
		fmt.Println("curl example:")
		fmt.Printf("curl -H 'Authorization: Bearer %s' http://localhost:8081/api/v1/posts\n", token[:40]+"...")

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
	}
}
