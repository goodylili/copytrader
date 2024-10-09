package database

import (
	"context"
	"gorm.io/gorm"
)

// BuyTransaction represents a record of a token purchase
type BuyTransaction struct {
	gorm.Model
	ContractAddress string  `gorm:"type:varchar(42);unique;not null"`
	TokenName       string  `gorm:"type:varchar(100);not null"`
	Ticker          string  `gorm:"type:varchar(10);not null"`
	EntryPrice      float64 `gorm:"not null"`
	TimeOfEntry     int64   `gorm:"not null"`
	Hash            string  `gorm:"type:varchar(66);unique;not null"`
}

// CreateBuyTransaction adds a new buy transaction to the database
func (d *Database) CreateBuyTransaction(ctx context.Context, txn BuyTransaction) error {
	return d.Client.WithContext(ctx).Create(&txn).Error
}

// GetBuyTransactionByCA retrieves a buy transaction by contract address
func (d *Database) GetBuyTransactionByCA(ctx context.Context, CA string) (BuyTransaction, error) {
	var txn BuyTransaction
	err := d.Client.WithContext(ctx).Where("ContractAddress = ?", CA).First(&txn).Error
	return txn, err
}

// GetBuyTransactionByHash retrieves a buy transaction by its hash
func (d *Database) GetBuyTransactionByHash(ctx context.Context, hash string) (BuyTransaction, error) {
	var txn BuyTransaction
	err := d.Client.WithContext(ctx).Where("Hash = ?", hash).First(&txn).Error
	return txn, err
}
