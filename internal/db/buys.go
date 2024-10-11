package database

import (
	"context"
	"gorm.io/gorm"
)

type BuyTransaction struct {
	gorm.Model
	ETHAmount       int
	ContractAddress string `gorm:"type:varchar(42);unique;not null"`
	Ticker          string `gorm:"type:varchar(10);not null"`
	Hash            string `gorm:"type:varchar(66);unique;not null"`
}

func (d *Database) CreateBuyTransaction(ctx context.Context, txn BuyTransaction) error {
	return d.Client.WithContext(ctx).Create(&txn).Error
}

func (d *Database) GetBuyTransactionByCA(ctx context.Context, CA string) (BuyTransaction, error) {
	var txn BuyTransaction
	err := d.Client.WithContext(ctx).Where("ContractAddress = ?", CA).First(&txn).Error
	return txn, err
}

func (d *Database) GetBuyTransactionByHash(ctx context.Context, hash string) (BuyTransaction, error) {
	var txn BuyTransaction
	err := d.Client.WithContext(ctx).Where("Hash = ?", hash).First(&txn).Error
	return txn, err
}
