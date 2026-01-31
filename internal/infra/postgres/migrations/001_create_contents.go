package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// createContentsTable creates the contents table with all indexes.
func createContentsTable() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "001_create_contents",
		Migrate: func(tx *gorm.DB) error {
			// Create table
			err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS contents (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					provider_id VARCHAR(50) NOT NULL,
					external_id VARCHAR(100) NOT NULL,
					title VARCHAR(500) NOT NULL,
					type VARCHAR(20) NOT NULL,
					tags TEXT[],
					
					-- Metrics
					views INTEGER DEFAULT 0,
					likes INTEGER DEFAULT 0,
					duration VARCHAR(20),
					reading_time INTEGER DEFAULT 0,
					reactions INTEGER DEFAULT 0,
					comments INTEGER DEFAULT 0,
					
					-- Score
					score DECIMAL(10,2) DEFAULT 0,
					
					-- Timestamps
					published_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					
					-- Unique constraint for upsert
					CONSTRAINT uq_provider_external UNIQUE (provider_id, external_id)
				);
			`).Error
			if err != nil {
				return err
			}

			// Create indexes
			indexes := []string{
				"CREATE INDEX IF NOT EXISTS idx_contents_type ON contents(type);",
				"CREATE INDEX IF NOT EXISTS idx_contents_score ON contents(score DESC);",
				"CREATE INDEX IF NOT EXISTS idx_contents_published_at ON contents(published_at DESC);",
				"CREATE INDEX IF NOT EXISTS idx_contents_provider_id ON contents(provider_id);",
			}

			for _, idx := range indexes {
				// Ignore error for trigram index if extension not installed
				_ = tx.Exec(idx).Error
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec("DROP TABLE IF EXISTS contents;").Error
		},
	}
}
