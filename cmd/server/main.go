package main

import (
	"copytrader/cmd"
	database "copytrader/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("Time to Copy Trade Those Wallets")
	configurations := cmd.LoadConfig()
	// Step 3: Set up the database connection
	db, err := database.NewDatabase(configurations.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
}
