// Package migrations provides database migrations using gormigrate.
package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// Migrations returns all database migrations.
func Migrations() []*gormigrate.Migration {
	return []*gormigrate.Migration{
		createContentsTable(),
		addFTSSupport(),
	}
}

// Run executes all pending migrations.
func Run(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, Migrations())
	return m.Migrate()
}

// Rollback rolls back the last migration.
func Rollback(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, Migrations())
	return m.RollbackLast()
}
