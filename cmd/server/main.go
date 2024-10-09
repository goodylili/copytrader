package main

import (
	"copytrader/config"
	database "copytrader/internal/db"
	"copytrader/internal/evm"
	"fmt"
	"log"
)

func main() {
	fmt.Println("Time to Copy Trade Those Wallets")
	configurations := config.LoadConfig()

	// Step 2: Set up the Ethereum client for Base network
	client, err := evm.NewClient(configurations.BaseRPC)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum client: %v", err)
	}
	// Step 3: Set up the database connection
	db, err := database.NewDatabase(configurations.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
}
