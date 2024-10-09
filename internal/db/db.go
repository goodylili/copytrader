package database

import (
	"context"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	Client *gorm.DB
}

func NewDatabase() (*Database, error) {
	// Hardcoded database file path
	dbPath := "./database.db"

	// Open the SQLite database
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Enable connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return &Database{
		Client: db,
	}, nil
}

// Ping - pings the database to check if it is alive
func (d *Database) Ping(ctx context.Context) error {
	client, err := d.Client.DB()
	if err != nil {
		return err
	}
	return client.PingContext(ctx)
}
