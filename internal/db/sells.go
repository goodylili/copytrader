package database

import (
	"context"
	"gorm.io/gorm"
)

// SellTransaction represents a record of a token sale, including PNL calculation
type SellTransaction struct {
	gorm.Model
	ContractAddress string  `gorm:"type:varchar(42);unique;not null"`
	ExitPrice       float64 `gorm:"not null"`
	TimeOfExit      int64   `gorm:"not null"`
	Hash            string  `gorm:"type:varchar(66);unique;not null"`
	PNL             float64 `gorm:"not null"` // PNL calculation
}

func (d *Database) CreateSellTransaction(ctx context.Context, txn SellTransaction) error {
	return d.Client.WithContext(ctx).Create(&txn).Error
}

// GetSellTransactionByCA retrieves a sell transaction by contract address
func (d *Database) GetSellTransactionByCA(ctx context.Context, CA string) (SellTransaction, error) {
	var txn SellTransaction
	err := d.Client.WithContext(ctx).Where("ContractAddress = ?", CA).First(&txn).Error
	return txn, err
}

// GetSellTransactionByHash retrieves a sell transaction by its hash
func (d *Database) GetSellTransactionByHash(ctx context.Context, hash string) (SellTransaction, error) {
	var txn SellTransaction
	err := d.Client.WithContext(ctx).Where("Hash = ?", hash).First(&txn).Error
	return txn, err
}
