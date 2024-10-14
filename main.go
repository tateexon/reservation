package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tateexon/reservation/api"
	"github.com/tateexon/reservation/db"
	"github.com/tateexon/reservation/schema"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var Version string

func main() {
	log.Println("Version:", Version)
	user := os.Getenv("POSTGRES_USER")
	if len(user) == 0 {
		log.Fatal("POSTGRES_USER not set")
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if len(password) == 0 {
		log.Fatal("POSTGRES_PASS not set")
	}
	url := os.Getenv("POSTGRES_URL")
	if len(url) == 0 {
		log.Fatal("POSTGRES_USER not set")
	}
	pdb := os.Getenv("POSTGRES_DB")
	if len(pdb) == 0 {
		log.Fatal("POSTGRES_DB not set")
	}

	if interval, ok := os.LookupEnv("AVAILABILITY_INTERVAL"); ok {
		_, err := time.ParseDuration(interval)
		if err != nil {
			log.Fatal("Invalid AVAILABILITY_INTERVAL set: ", err)
		}
	}

	// Database connection string
	// connStr := "postgres://youruser:yourpassword@localhost:5432/yourdb?sslmode=disable"
	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, url, pdb)

	// Initialize database
	database, err := db.NewDatabase(connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize server
	server := &api.Server{DB: database}

	// Set up Gin router
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		// Allow all origins (you can restrict this to specific origins)
		AllowOrigins: []string{"*"},
		// Allow specific HTTP methods
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// Allow specific HTTP headers
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		// Expose headers to the browser
		ExposeHeaders: []string{"Content-Length"},
		// Allow credentials (cookies, authorization headers, etc.)
		AllowCredentials: true,
		// Max age for caching preflight responses
		MaxAge: 12 * time.Hour,
	}))

	// Register handlers
	schema.RegisterHandlers(router, server)

	// Run the server
	err = router.Run(":8080")
	log.Fatal("Failed to run the server", err)
}
