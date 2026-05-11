package migrations

import (
	"log"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) {
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		M20260506CreateExecutions(),
	})
	if err := m.Migrate(); err != nil {
		log.Fatalf("Could not run PostgreSQL migrations: %v", err)
	}
	log.Println("PostgreSQL migrations ran successfully")
}
