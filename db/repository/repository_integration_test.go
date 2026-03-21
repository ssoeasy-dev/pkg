//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	. "github.com/ssoeasy-dev/pkg/db/repository"
)

// ─── Модели ───────────────────────────────────────────────────────────────────

type Article struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title     string    `gorm:"not null"`
	Author    string    `gorm:"not null"`
	Views     int       `gorm:"default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
	// gorm.DeletedAt (не *time.Time) активирует soft-delete GORM:
	// автоматически добавляет WHERE deleted_at IS NULL и
	// генерирует UPDATE вместо DELETE при мягком удалении.
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Article) TableName() string { return "articles" }

// ─── Setup ────────────────────────────────────────────────────────────────────

func setupPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	ctx := context.Background()

	pgContainer, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&Article{}))
	return db
}

func newTestLogger() *logger.Logger {
	return logger.NewLogger(logger.EnvironmentTest, "test")
}

func newRepo(db *gorm.DB) Repository[Article] {
	return NewRepository[Article](tx.NewTxManager(db), newTestLogger(), "article")
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestRepository_Create(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	a := &Article{Title: "Hello", Author: "Alice"}
	require.NoError(t, repo.Create(ctx, a))
	assert.NotEqual(t, uuid.Nil, a.ID)
}

func TestRepository_Create_TwoDuplicatesAllowed(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Article{Title: "Same", Author: "A"}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "Same", Author: "B"}))
}

// ─── FindOne ──────────────────────────────────────────────────────────────────

func TestRepository_FindOne(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	a := &Article{Title: "Find me", Author: "Bob"}
	require.NoError(t, repo.Create(ctx, a))

	found, err := repo.FindOne(ctx, WithConditions(map[string]any{"id": a.ID}))
	require.NoError(t, err)
	assert.Equal(t, a.ID, found.ID)
	assert.Equal(t, "Find me", found.Title)
}

func TestRepository_FindOne_NotFound(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	_, err := repo.FindOne(ctx, WithConditions(map[string]any{"id": uuid.New()}))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_FindOne_WithSelect(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	a := &Article{Title: "Partial", Author: "Carol"}
	require.NoError(t, repo.Create(ctx, a))

	found, err := repo.FindOne(ctx,
		WithConditions(map[string]any{"id": a.ID}),
		WithSelect("id", "title"),
	)
	require.NoError(t, err)
	assert.Equal(t, "Partial", found.Title)
	assert.Empty(t, found.Author) // не выбирали
}

// ─── FindAll ──────────────────────────────────────────────────────────────────

func TestRepository_FindAll(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	for i := range 5 {
		require.NoError(t, repo.Create(ctx, &Article{
			Title:  fmt.Sprintf("Article %d", i),
			Author: "Dave",
		}))
	}

	all, err := repo.FindAll(ctx, WithConditions(map[string]any{"author": "Dave"}))
	require.NoError(t, err)
	assert.Len(t, all, 5)
}

func TestRepository_FindAll_WithPagination(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	for i := range 10 {
		require.NoError(t, repo.Create(ctx, &Article{
			Title:  fmt.Sprintf("Page-%02d", i),
			Author: "Eve",
		}))
	}

	page1, err := repo.FindAll(ctx,
		WithConditions(map[string]any{"author": "Eve"}),
		WithPagination(Pagination{Limit: 3, Page: 1}),
		WithOrder(Order{By: "title", Dir: OrderDirAsc}),
	)
	require.NoError(t, err)
	require.Len(t, page1, 3)

	page2, err := repo.FindAll(ctx,
		WithConditions(map[string]any{"author": "Eve"}),
		WithPagination(Pagination{Limit: 3, Page: 2}),
		WithOrder(Order{By: "title", Dir: OrderDirAsc}),
	)
	require.NoError(t, err)
	require.Len(t, page2, 3)
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestRepository_FindAll_OrderASC(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	for _, title := range []string{"Zebra", "Apple", "Mango"} {
		require.NoError(t, repo.Create(ctx, &Article{Title: title, Author: "Frank"}))
	}

	results, err := repo.FindAll(ctx,
		WithConditions(map[string]any{"author": "Frank"}),
		WithOrder(Order{By: "title", Dir: OrderDirAsc}),
	)
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "Apple", results[0].Title)
	assert.Equal(t, "Zebra", results[2].Title)
}

func TestRepository_FindAll_OR(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Article{Title: "A", Author: "Grace"}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "B", Author: "Henry"}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "C", Author: "Other"}))

	// author = 'Grace' OR author = 'Henry' → 2 записи, 'Other' не попадает
	results, err := repo.FindAll(ctx,
		WithConditions(
			map[string]any{"author": "Grace"},
			map[string]any{"author": "Henry"},
		),
	)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestRepository_FindAll_ORDoesNotMixFields(t *testing.T) {
	// Проверяем корректность OR-группировки:
	// (author='Iris' AND views=0) OR (author='Jack' AND views=99)
	// НЕ должна вернуть запись с author='Iris' AND views=99
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Article{Title: "I0", Author: "Iris", Views: 0}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "I99", Author: "Iris", Views: 99}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "J99", Author: "Jack", Views: 99}))

	results, err := repo.FindAll(ctx,
		WithConditions(
			map[string]any{"author": "Iris", "views": 0},
			map[string]any{"author": "Jack", "views": 99},
		),
	)
	require.NoError(t, err)
	require.Len(t, results, 2)

	titles := []string{results[0].Title, results[1].Title}
	assert.Contains(t, titles, "I0")
	assert.Contains(t, titles, "J99")
	assert.NotContains(t, titles, "I99") // Iris+99 не должна попасть
}

func TestRepository_FindAll_Like(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Article{Title: "golang tips", Author: "Iris"}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "python tips", Author: "Iris"}))
	require.NoError(t, repo.Create(ctx, &Article{Title: "golang patterns", Author: "Iris"}))

	results, err := repo.FindAll(ctx,
		WithConditions(map[string]any{"title": Like("golang%")}),
	)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestRepository_FindAll_IN(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	var ids []uuid.UUID
	for _, title := range []string{"X", "Y", "Z"} {
		a := &Article{Title: title, Author: "Jack"}
		require.NoError(t, repo.Create(ctx, a))
		if title != "Z" {
			ids = append(ids, a.ID)
		}
	}

	results, err := repo.FindAll(ctx,
		WithConditions(map[string]any{"id": ids}),
	)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// ─── Count / Exists ───────────────────────────────────────────────────────────

func TestRepository_Count(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	for range 4 {
		require.NoError(t, repo.Create(ctx, &Article{Title: "T", Author: "Kate"}))
	}

	count, err := repo.Count(ctx, WithConditions(map[string]any{"author": "Kate"}))
	require.NoError(t, err)
	assert.EqualValues(t, 4, count)
}

func TestRepository_Exists_True(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Article{Title: "Exists", Author: "Leo"}))

	ok, err := repo.Exists(ctx, WithConditions(map[string]any{"author": "Leo"}))
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRepository_Exists_False(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	ok, err := repo.Exists(ctx, WithConditions(map[string]any{"author": "Nobody"}))
	require.NoError(t, err)
	assert.False(t, ok)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestRepository_Update(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	a := &Article{Title: "Before", Author: "Mia"}
	require.NoError(t, repo.Create(ctx, a))

	n, err := repo.Update(ctx,
		map[string]any{"title": "After"},
		WithConditions(map[string]any{"id": a.ID}),
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	found, err := repo.FindOne(ctx, WithConditions(map[string]any{"id": a.ID}))
	require.NoError(t, err)
	assert.Equal(t, "After", found.Title)
}

func TestRepository_Update_ZeroRowsNotError(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	n, err := repo.Update(ctx,
		map[string]any{"title": "Ghost"},
		WithConditions(map[string]any{"id": uuid.New()}),
	)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

func TestRepository_Update_Multiple(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	for range 3 {
		require.NoError(t, repo.Create(ctx, &Article{Title: "Bulk", Author: "Ned"}))
	}

	n, err := repo.Update(ctx,
		map[string]any{"views": 100},
		WithConditions(map[string]any{"author": "Ned"}),
	)
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestRepository_Delete_Soft(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	a := &Article{Title: "Soft", Author: "Olivia"}
	require.NoError(t, repo.Create(ctx, a))
	require.NotEqual(t, uuid.Nil, a.ID, "Create must populate ID via gen_random_uuid()")

	n, err := repo.Delete(ctx, false, WithConditions(map[string]any{"id": a.ID}))
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	// Soft-delete: обычный FindOne не находит (deleted_at IS NULL фильтр)
	_, err = repo.FindOne(ctx, WithConditions(map[string]any{"id": a.ID}))
	assert.ErrorIs(t, err, ErrNotFound)

	// WithDeleted(true) (Unscoped) находит мягко удалённую запись
	found, err := repo.FindOne(ctx,
		WithConditions(map[string]any{"id": a.ID}),
		WithDeleted(true),
	)
	require.NoError(t, err)
	assert.Equal(t, a.ID, found.ID)
	assert.True(t, found.DeletedAt.Valid, "deleted_at должен быть установлен")
}

func TestRepository_Delete_Hard(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	a := &Article{Title: "Hard", Author: "Pete"}
	require.NoError(t, repo.Create(ctx, a))

	n, err := repo.Delete(ctx, true, WithConditions(map[string]any{"id": a.ID}))
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)

	// Hard-delete: запись не найти даже с WithDeleted(true)
	_, err = repo.FindOne(ctx,
		WithConditions(map[string]any{"id": a.ID}),
		WithDeleted(true),
	)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_Delete_ZeroRowsNotError(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	n, err := repo.Delete(ctx, false, WithConditions(map[string]any{"id": uuid.New()}))
	require.NoError(t, err)
	assert.EqualValues(t, 0, n)
}

// ─── RawQuery ─────────────────────────────────────────────────────────────────

func TestRepository_RawQuery(t *testing.T) {
	repo := newRepo(setupPostgres(t))
	ctx := context.Background()

	for _, author := range []string{"Quinn", "Quinn", "Ray"} {
		require.NoError(t, repo.Create(ctx, &Article{Title: "T", Author: author}))
	}

	results, err := repo.RawQuery(ctx,
		`SELECT * FROM articles WHERE author = ? AND deleted_at IS NULL`,
		"Quinn",
	)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// ─── Транзакции ───────────────────────────────────────────────────────────────

func TestRepository_Transaction_Commit(t *testing.T) {
	db := setupPostgres(t)
	txMgr := tx.NewTxManager(db)
	ctx := context.Background()

	err := txMgr.WithTransaction(ctx, func(ctx context.Context) error {
		repo := NewRepository[Article](txMgr, newTestLogger(), "article")
		return repo.Create(ctx, &Article{Title: "Tx Commit", Author: "Sam"})
	})
	require.NoError(t, err)

	repo := newRepo(db)
	ok, err := repo.Exists(ctx, WithConditions(map[string]any{"author": "Sam"}))
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRepository_Transaction_Rollback(t *testing.T) {
	db := setupPostgres(t)
	txMgr := tx.NewTxManager(db)
	ctx := context.Background()

	err := txMgr.WithTransaction(ctx, func(ctx context.Context) error {
		repo := NewRepository[Article](txMgr, newTestLogger(), "article")
		_ = repo.Create(ctx, &Article{Title: "Will rollback", Author: "Tina"})
		return fmt.Errorf("intentional error")
	})
	require.Error(t, err)

	repo := newRepo(db)
	ok, err := repo.Exists(ctx, WithConditions(map[string]any{"author": "Tina"}))
	require.NoError(t, err)
	assert.False(t, ok)
}
