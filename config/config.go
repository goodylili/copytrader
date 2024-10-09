package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PrivateKey         string
	PublicKey          string
	ChainID            string
	BaseRPC            string
	UniswapBaseRouter  string
	UniswapBaseFactory string
	WethBaseAddress    string
	Redis              string
}

func LoadConfig() *Config {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Return the configuration struct populated with environment variables
	return &Config{
		PrivateKey:         os.Getenv("PRIVATE_KEY"),
		PublicKey:          os.Getenv("PUBLIC_KEY"),
		ChainID:            os.Getenv("CHAIN_ID"),
		Redis:              os.Getenv("REDIS"),
		BaseRPC:            os.Getenv("BASE_RPC"),
		UniswapBaseRouter:  os.Getenv("UNISWAP_BASE_ROUTER"),
		UniswapBaseFactory: os.Getenv("UNISWAP_BASE_FACTORY"),
		WethBaseAddress:    os.Getenv("WETH_BASE_ADDRESS"),
	}
}
