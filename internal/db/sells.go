package database

import (
	"context"
	"gorm.io/gorm"
)

type SellTransaction struct {
	gorm.Model
	ContractAddress string  `gorm:"type:varchar(42);unique;not null"`
	ETHReceived     float64 `gorm:"not null"`
	Hash            string  `gorm:"type:varchar(66);unique;not null"`
}

func (d *Database) CreateSellTransaction(ctx context.Context, txn SellTransaction) error {
	return d.Client.WithContext(ctx).Create(&txn).Error
}

func (d *Database) GetSellTransactionByCA(ctx context.Context, CA string) (SellTransaction, error) {
	var txn SellTransaction
	err := d.Client.WithContext(ctx).Where("ContractAddress = ?", CA).First(&txn).Error
	return txn, err
}

func (d *Database) GetSellTransactionByHash(ctx context.Context, hash string) (SellTransaction, error) {
	var txn SellTransaction
	err := d.Client.WithContext(ctx).Where("Hash = ?", hash).First(&txn).Error
	return txn, err
}
