package postgres

import (
	"context"
	"search-engine-service/internal/domain"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgresContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDB creates a PostgreSQL testcontainer and returns a connected GORM DB
//
// Prerequisites:
//   - Docker must be running
//   - Run: docker-compose up postgres
//
// OR
//   - Skip tests with: go test -short
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := postgresContainer.Run(ctx,
		"postgres:16-alpine",
		postgresContainer.WithDatabase("testdb"),
		postgresContainer.WithUsername("testuser"),
		postgresContainer.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf(`Failed to start PostgreSQL container: %v

Docker Prerequisites:
1. Ensure Docker is running
2. OR use existing postgres: docker-compose up postgres
3. OR skip integration tests: go test -short

`, err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "Failed to get connection string")

	// Connect to database
	db, err := gorm.Open(postgresDriver.Open(connStr), &gorm.Config{
		Logger: nil, // Silent logger for tests
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations
	err = db.AutoMigrate(&ContentModel{})
	require.NoError(t, err, "Failed to run migrations")

	// Cleanup function
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

// createTestContent is a factory function for creating test content
func createTestContent(providerID, externalID string) *domain.Content {
	return &domain.Content{
		ProviderID:  providerID,
		ExternalID:  externalID,
		Title:       "Test Title",
		Type:        "article",
		Tags:        []string{"tag1", "tag2"},
		Views:       100,
		Likes:       10,
		Duration:    "",
		ReadingTime: 5,
		Reactions:   0,
		Comments:    0,
		Score:       75.5,
		PublishedAt: time.Now().UTC(),
	}
}

// TestUpsert_InsertNew verifies that Upsert creates a new record
func TestUpsert_InsertNew(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create new content
	content := createTestContent("provider_a", "ext_123")

	// Execute upsert
	err := repo.Upsert(ctx, content)
	require.NoError(t, err)

	// Verify record was created
	assert.NotEmpty(t, content.ID, "ID should be generated")
	assert.False(t, content.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, content.UpdatedAt.IsZero(), "UpdatedAt should be set")

	// Verify record exists in database
	var model ContentModel
	err = db.Where("provider_id = ? AND external_id = ?", "provider_a", "ext_123").First(&model).Error
	require.NoError(t, err)
	assert.Equal(t, content.ID, model.ID)
	assert.Equal(t, "Test Title", model.Title)
}

// TestUpsert_UpdateExisting verifies that Upsert updates an existing record
func TestUpsert_UpdateExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Insert initial content
	content := createTestContent("provider_a", "ext_123")
	err := repo.Upsert(ctx, content)
	require.NoError(t, err)

	// Capture original values
	originalID := content.ID
	originalCreatedAt := content.CreatedAt
	originalUpdatedAt := content.UpdatedAt

	// Wait to ensure UpdatedAt will be different
	time.Sleep(10 * time.Millisecond)

	// Update content
	content.Title = "Updated Title"
	content.Views = 200
	err = repo.Upsert(ctx, content)
	require.NoError(t, err)

	// Verify ID unchanged
	assert.Equal(t, originalID, content.ID, "ID should remain unchanged")

	// Verify CreatedAt unchanged
	assert.Equal(t, originalCreatedAt.Unix(), content.CreatedAt.Unix(), "CreatedAt should remain unchanged")

	// Verify UpdatedAt changed
	assert.True(t, content.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be newer")

	// Verify updates persisted
	var model ContentModel
	err = db.Where("id = ?", content.ID).First(&model).Error
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", model.Title)
	assert.Equal(t, 200, model.Views)
}

// TestBulkUpsert_MixedOperations verifies BulkUpsert handles mixed new and existing records
func TestBulkUpsert_MixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Insert 2 existing contents
	existing1 := createTestContent("provider_a", "ext_001")
	existing2 := createTestContent("provider_a", "ext_002")
	err := repo.Upsert(ctx, existing1)
	require.NoError(t, err)
	err = repo.Upsert(ctx, existing2)
	require.NoError(t, err)

	// Capture original IDs
	id1 := existing1.ID
	id2 := existing2.ID

	// Prepare bulk upsert: 2 updates + 3 new
	contents := []*domain.Content{
		// Updates
		{
			ProviderID:  "provider_a",
			ExternalID:  "ext_001",
			Title:       "Updated Title 1",
			Type:        "article",
			Tags:        []string{"new"},
			Views:       500,
			PublishedAt: time.Now().UTC(),
		},
		{
			ProviderID:  "provider_a",
			ExternalID:  "ext_002",
			Title:       "Updated Title 2",
			Type:        "video",
			Tags:        []string{"new"},
			Views:       600,
			PublishedAt: time.Now().UTC(),
		},
		// New records
		createTestContent("provider_a", "ext_003"),
		createTestContent("provider_b", "ext_004"),
		createTestContent("provider_b", "ext_005"),
	}

	// Execute bulk upsert
	err = repo.BulkUpsert(ctx, contents)
	require.NoError(t, err)

	// Verify total count
	var count int64
	err = db.Model(&ContentModel{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(5), count, "Should have exactly 5 records")

	// Verify existing IDs unchanged
	assert.Equal(t, id1, contents[0].ID, "First ID should remain unchanged")
	assert.Equal(t, id2, contents[1].ID, "Second ID should remain unchanged")

	// Verify new IDs generated
	assert.NotEmpty(t, contents[2].ID, "Third ID should be generated")
	assert.NotEmpty(t, contents[3].ID, "Fourth ID should be generated")
	assert.NotEmpty(t, contents[4].ID, "Fifth ID should be generated")

	// Verify updates persisted
	var model ContentModel
	err = db.Where("id = ?", id1).First(&model).Error
	require.NoError(t, err)
	assert.Equal(t, "Updated Title 1", model.Title)
	assert.Equal(t, 500, model.Views)

	// Verify all UpdatedAt timestamps are recent (within last minute)
	for i, content := range contents {
		assert.WithinDuration(t, time.Now(), content.UpdatedAt, time.Minute,
			"Content %d UpdatedAt should be recent", i)
	}
}

// TestUpsert_TimestampVerification verifies UpdatedAt changes on update
func TestUpsert_TimestampVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Insert initial content
	content := createTestContent("provider_a", "ext_123")
	err := repo.Upsert(ctx, content)
	require.NoError(t, err)

	createdAt := content.CreatedAt
	updatedAt := content.UpdatedAt

	// Wait to ensure timestamp difference
	time.Sleep(100 * time.Millisecond)

	// Update content
	content.Title = "Updated Title"
	err = repo.Upsert(ctx, content)
	require.NoError(t, err)

	// Verify timestamps
	assert.True(t, content.UpdatedAt.After(updatedAt), "UpdatedAt should be newer after update")
	assert.True(t, content.UpdatedAt.After(createdAt), "UpdatedAt should be after CreatedAt")
	assert.Equal(t, createdAt.Unix(), content.CreatedAt.Unix(), "CreatedAt should not change")

	// Verify timestamp is recent
	assert.WithinDuration(t, time.Now(), content.UpdatedAt, time.Minute,
		"UpdatedAt should be within last minute")
}

// TestUpsert_PrimaryKeyStability verifies ID doesn't change across multiple updates
func TestUpsert_PrimaryKeyStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Insert initial content
	content := createTestContent("provider_a", "ext_123")
	err := repo.Upsert(ctx, content)
	require.NoError(t, err)

	originalID := content.ID
	assert.NotEmpty(t, originalID, "Original ID should be generated")

	// Perform 3 updates with different data
	updates := []struct {
		title string
		views int
	}{
		{"First Update", 100},
		{"Second Update", 200},
		{"Third Update", 300},
	}

	for i, update := range updates {
		content.Title = update.title
		content.Views = update.views
		err = repo.Upsert(ctx, content)
		require.NoError(t, err, "Update %d should succeed", i+1)

		// Verify ID hasn't changed
		assert.Equal(t, originalID, content.ID,
			"ID should remain unchanged after update %d", i+1)
	}

	// Final verification from database
	var model ContentModel
	err = db.Where("provider_id = ? AND external_id = ?", "provider_a", "ext_123").First(&model).Error
	require.NoError(t, err)
	assert.Equal(t, originalID, model.ID, "Database ID should match original")
	assert.Equal(t, "Third Update", model.Title, "Should have latest update")
	assert.Equal(t, 300, model.Views, "Should have latest views")
}

// TestUpsert_ConcurrentOperations verifies goroutine safety
func TestUpsert_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	errChan := make(chan error, goroutines)

	// Launch goroutines that all upsert the same provider_id + external_id
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()

			content := &domain.Content{
				ProviderID:  "provider_a",
				ExternalID:  "concurrent_test",
				Title:       "Concurrent Title " + string(rune('A'+iteration)),
				Type:        "article",
				Tags:        []string{"concurrent"},
				Views:       iteration * 100,
				PublishedAt: time.Now().UTC(),
			}

			if err := repo.Upsert(ctx, content); err != nil {
				errChan <- err
			}
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	assert.Empty(t, errors, "No errors should occur during concurrent upserts")

	// Verify exactly 1 record exists
	var count int64
	err := db.Model(&ContentModel{}).
		Where("provider_id = ? AND external_id = ?", "provider_a", "concurrent_test").
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "Should have exactly 1 record after concurrent upserts")

	// Verify final state is valid
	var model ContentModel
	err = db.Where("provider_id = ? AND external_id = ?", "provider_a", "concurrent_test").
		First(&model).Error
	require.NoError(t, err)
	assert.NotEmpty(t, model.ID, "Should have valid ID")
	assert.NotEmpty(t, model.Title, "Should have a title")
	assert.Equal(t, "article", model.Type, "Should have correct type")
}

// TestBulkUpsert_EmptySlice verifies handling of empty input
func TestBulkUpsert_EmptySlice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Test with empty slice
	err := repo.BulkUpsert(ctx, []*domain.Content{})
	assert.NoError(t, err, "Empty slice should not cause error")

	// Test with nil slice
	err = repo.BulkUpsert(ctx, nil)
	assert.NoError(t, err, "Nil slice should not cause error")
}

// TestBulkUpsert_LargeBatch verifies batch processing with large datasets
func TestBulkUpsert_LargeBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Create 500 test records to verify batch size of 100
	const recordCount = 500
	contents := make([]*domain.Content, recordCount)
	for i := 0; i < recordCount; i++ {
		contents[i] = createTestContent("provider_a", "ext_"+string(rune('A'+i/26))+string(rune('A'+i%26)))
	}

	// Execute bulk upsert
	err := repo.BulkUpsert(ctx, contents)
	require.NoError(t, err)

	// Verify all records created
	var count int64
	err = db.Model(&ContentModel{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(recordCount), count, "Should have created all records")

	// Verify all IDs generated
	for i, content := range contents {
		assert.NotEmpty(t, content.ID, "Content %d should have ID", i)
	}
}

// TestUpsert_UniqueConstraintEnforced verifies the composite unique constraint works
func TestUpsert_UniqueConstraintEnforced(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	// Insert first record
	content1 := createTestContent("provider_a", "ext_123")
	err := repo.Upsert(ctx, content1)
	require.NoError(t, err)

	id1 := content1.ID

	// Upsert same provider_id + external_id with different data
	content2 := createTestContent("provider_a", "ext_123")
	content2.Title = "Different Title"
	err = repo.Upsert(ctx, content2)
	require.NoError(t, err)

	// Should have updated existing record, not created new one
	assert.Equal(t, id1, content2.ID, "Should reuse same ID (update, not insert)")

	// Verify only 1 record exists
	var count int64
	err = db.Model(&ContentModel{}).
		Where("provider_id = ? AND external_id = ?", "provider_a", "ext_123").
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "Should have exactly 1 record")

	// Verify title was updated
	var model ContentModel
	err = db.Where("id = ?", id1).First(&model).Error
	require.NoError(t, err)
	assert.Equal(t, "Different Title", model.Title)
}
