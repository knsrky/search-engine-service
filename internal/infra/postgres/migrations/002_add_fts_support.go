package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// addFTSSupport implements PostgreSQL Full-Text Search with hybrid ranking.
//
// ## What This Migration Does
//
// 1. Adds `search_vector` column (tsvector type) for FTS indexing
// 2. Creates GIN index for O(log n) full-text search performance
// 3. Creates trigger function to auto-update search_vector on INSERT/UPDATE
// 4. Populates existing rows with search vectors
//
// ## Search Vector Weights
//
// - Title: Weight 'A' (highest priority, ~4x multiplier)
// - Tags: Weight 'B' (medium priority, ~2x multiplier)
//
// ## Hybrid Ranking Formula
//
// When sorting by relevance:
//
//	final_rank = ts_rank(search_vector, query) × LOG(score + 10)
//
// Components:
// - ts_rank: Text relevance (0-1), measures how well content matches query
// - LOG(score + 10): Logarithmic popularity normalization
//   - LOG prevents viral content from dominating (1M views → 6, not 1M)
//   - +10 smoothing ensures new content (score=0) gets baseline rank
//
// ## Why Multiply (Not Add)?
//
// Multiplication gives text relevance "veto power":
// - Irrelevant content (ts_rank=0) always gets 0, regardless of popularity
// - This prevents popular-but-irrelevant items from polluting search results
func addFTSSupport() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "002_add_fts_support",
		Migrate: func(tx *gorm.DB) error {
			// 1. Add search_vector column
			if err := tx.Exec(`
				ALTER TABLE contents
				ADD COLUMN IF NOT EXISTS search_vector tsvector
			`).Error; err != nil {
				return err
			}

			// 2. Create GIN index for fast FTS queries
			if err := tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_contents_search_vector
				ON contents USING GIN (search_vector)
			`).Error; err != nil {
				return err
			}

			// 3. Create trigger function for auto-updating search_vector
			if err := tx.Exec(`
				CREATE OR REPLACE FUNCTION contents_search_vector_update()
				RETURNS trigger AS $$
				BEGIN
					NEW.search_vector :=
						setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
						setweight(to_tsvector('english', coalesce(array_to_string(NEW.tags, ' '), '')), 'B');
					RETURN NEW;
				END
				$$ LANGUAGE plpgsql
			`).Error; err != nil {
				return err
			}

			// 4. Create trigger on INSERT/UPDATE
			if err := tx.Exec(`
    DROP TRIGGER IF EXISTS trg_contents_search_vector ON contents
`).Error; err != nil {
				return err
			}

			if err := tx.Exec(`
    CREATE TRIGGER trg_contents_search_vector
    BEFORE INSERT OR UPDATE OF title, tags
    ON contents
    FOR EACH ROW
    EXECUTE FUNCTION contents_search_vector_update()
`).Error; err != nil {
				return err
			}

			// 5. Populate existing rows
			if err := tx.Exec(`
				UPDATE contents SET search_vector =
					setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
					setweight(to_tsvector('english', coalesce(array_to_string(tags, ' '), '')), 'B')
				WHERE search_vector IS NULL
			`).Error; err != nil {
				return err
			}

			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			_ = tx.Exec(`DROP TRIGGER IF EXISTS trg_contents_search_vector ON contents`).Error
			_ = tx.Exec(`DROP FUNCTION IF EXISTS contents_search_vector_update()`).Error
			_ = tx.Exec(`DROP INDEX IF EXISTS idx_contents_search_vector`).Error
			_ = tx.Exec(`ALTER TABLE contents DROP COLUMN IF EXISTS search_vector`).Error
			return nil
		},
	}
}
