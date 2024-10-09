package evm

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

type Ethereum struct {
	Client *ethclient.Client
}

func NewClient(rpcURL string) (*Ethereum, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Printf("Failed to connect to the Ethereum client: %v", err)
		return nil, err
	}

	return &Ethereum{
		Client: client,
	}, nil
}

func (e *Ethereum) Ping() error {
	ctx := context.Background()

	_, err := e.Client.BlockNumber(ctx)
	if err != nil {
		log.Printf("Ping failed: %v", err)
		return fmt.Errorf("failed to ping Ethereum node: %w", err)
	}
	log.Println("Successfully connected to the Ethereum blockchain")
	return nil
}
