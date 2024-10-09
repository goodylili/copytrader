package database

import "log"

func (d *Database) MigrateDB() error {
	log.Println("Database Migration in Process...")
	err := d.Client.AutoMigrate(&BuyTransaction{}, &SellTransaction{})
	if err != nil {
		return err
	}
	log.Println("Database Migration Complete!")
	return nil
}
