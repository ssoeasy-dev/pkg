package repository_test

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	. "github.com/ssoeasy-dev/pkg/db/repository"
)

// testModel — упрощённая модель для проверки SQL
type testModel struct {
	ID    uint   `gorm:"primaryKey"`
	Login string `gorm:"column:login"`
	Email string `gorm:"column:email"`
	Views int    `gorm:"column:views"`
}

func (testModel) TableName() string { return "test_models" }

// buildSQL применяет опции к DryRun-сессии и возвращает итоговый SQL.
// Использует in-memory SQLite только для диалекта — реальных запросов нет.
func buildSQL(t *testing.T, opts ...RepositoryOption) string {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DryRun: true,
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	tx := db.Session(&gorm.Session{NewDB: true, DryRun: true}).
		Model(&testModel{})
	require.NoError(t, tx.Statement.Parse(&testModel{}))

	for _, opt := range opts {
		tx = opt(tx)
	}

	stmt := tx.Find(&[]testModel{}).Statement
	return stmt.SQL.String()
}

// ─── WithConditions ───────────────────────────────────────────────────────────

func TestWithConditions_SimpleField(t *testing.T) {
	sql := buildSQL(t, WithConditions(map[string]any{"login": "john"}))
	// Поле должно быть квалифицировано именем таблицы.
	// Форма кавычек зависит от диалекта (SQLite: `table`.`col`, PG: "table"."col"),
	// поэтому проверяем только наличие table.field без кавычек.
	assert.Contains(t, sql, "test_models.login")
	assert.Contains(t, sql, "?")
}

func TestWithConditions_MultipleFieldsAND(t *testing.T) {
	sql := buildSQL(t,
		WithConditions(map[string]any{"login": "john", "views": 5}),
	)
	// Оба поля присутствуют
	assert.Contains(t, sql, "login")
	assert.Contains(t, sql, "views")
	// AND внутри одной map — нет OR
	assert.NotContains(t, strings.ToUpper(sql), " OR ")
}

func TestWithConditions_TwoMapsProduceOR(t *testing.T) {
	sql := buildSQL(t,
		WithConditions(
			map[string]any{"login": "alice"},
			map[string]any{"login": "bob"},
		),
	)
	assert.Contains(t, strings.ToUpper(sql), " OR ")
}

func TestWithConditions_ORGroupingIsCorrect(t *testing.T) {
	// (login='alice' AND views=0) OR (login='bob' AND views=99)
	// SQL должен содержать оба условия в правильной группировке
	sql := buildSQL(t,
		WithConditions(
			map[string]any{"login": "alice", "views": 0},
			map[string]any{"login": "bob", "views": 99},
		),
	)
	assert.Contains(t, strings.ToUpper(sql), " OR ")
	assert.Contains(t, sql, "login")
	assert.Contains(t, sql, "views")
}

func TestWithConditions_Like(t *testing.T) {
	sql := buildSQL(t, WithConditions(map[string]any{"login": Like("%john%")}))
	assert.Contains(t, strings.ToUpper(sql), "LIKE")
}

func TestWithConditions_IsNullTrue(t *testing.T) {
	sql := buildSQL(t, WithConditions(map[string]any{"email": IsNull(true)}))
	assert.Contains(t, strings.ToUpper(sql), "IS NULL")
	assert.NotContains(t, strings.ToUpper(sql), "IS NOT NULL")
}

func TestWithConditions_IsNullFalse(t *testing.T) {
	sql := buildSQL(t, WithConditions(map[string]any{"email": IsNull(false)}))
	assert.Contains(t, strings.ToUpper(sql), "IS NOT NULL")
}

func TestWithConditions_SliceProducesIN(t *testing.T) {
	sql := buildSQL(t, WithConditions(map[string]any{"id": []uint{1, 2, 3}}))
	assert.Contains(t, strings.ToUpper(sql), "IN")
}

func TestWithConditions_EmptyMapSkipped(t *testing.T) {
	sql := buildSQL(t, WithConditions(map[string]any{}))
	assert.NotContains(t, strings.ToUpper(sql), "WHERE")
}

func TestWithConditions_AlreadyQualifiedFieldNotDoubled(t *testing.T) {
	// Поле с точкой не получает дополнительный префикс таблицы.
	// SQLite квотирует идентификаторы бэктиками: `other_table`.`id`
	sql := buildSQL(t, WithConditions(map[string]any{"other_table.id": 1}))
	assert.NotContains(t, sql, "test_models.`other_table`")
	assert.NotContains(t, sql, "test_models.other_table")
	assert.Contains(t, sql, "other_table")
}

func TestWithConditions_FieldWithSpaceNotQualified(t *testing.T) {
	// Поле с пробелом — сырое выражение, не квалифицируется
	sql := buildSQL(t, WithConditions(map[string]any{"login IS NOT NULL OR 1": "1"}))
	assert.NotContains(t, sql, "test_models.login IS NOT NULL")
}

// ─── WithOrder ────────────────────────────────────────────────────────────────

func TestWithOrder_ASC(t *testing.T) {
	sql := buildSQL(t, WithOrder(Order{By: "login", Dir: OrderDirAsc}))
	assert.Contains(t, strings.ToUpper(sql), "ORDER BY")
	assert.Contains(t, sql, "asc")
}

func TestWithOrder_DESC(t *testing.T) {
	sql := buildSQL(t, WithOrder(Order{By: "login", Dir: OrderDirDesc}))
	assert.Contains(t, sql, "desc")
}

func TestWithOrder_DefaultsToASC(t *testing.T) {
	sql := buildSQL(t, WithOrder(Order{By: "login"}))
	assert.Contains(t, sql, "asc")
}

func TestWithOrder_MultipleFields(t *testing.T) {
	sql := buildSQL(t,
		WithOrder(
			Order{By: "login", Dir: OrderDirAsc},
			Order{By: "views", Dir: OrderDirDesc},
		),
	)
	assert.Contains(t, sql, "login")
	assert.Contains(t, sql, "views")
}

func TestWithOrder_QualifiedFieldNotDoubled(t *testing.T) {
	sql := buildSQL(t, WithOrder(Order{By: "other.created_at", Dir: OrderDirDesc}))
	assert.NotContains(t, sql, "test_models.other.created_at")
	assert.Contains(t, sql, "other.created_at")
}

func TestWithOrder_EmptyByIsSkipped(t *testing.T) {
	sql := buildSQL(t, WithOrder(Order{By: ""}))
	assert.NotContains(t, strings.ToUpper(sql), "ORDER BY")
}

// ─── WithSelect ───────────────────────────────────────────────────────────────

func TestWithSelect_SimpleFieldsQualified(t *testing.T) {
	sql := buildSQL(t, WithSelect("login", "email"))
	assert.Contains(t, sql, "login")
	assert.Contains(t, sql, "email")
}

func TestWithSelect_QualifiedFieldPreserved(t *testing.T) {
	sql := buildSQL(t, WithSelect("other.id"))
	assert.Contains(t, sql, "other.id")
	assert.NotContains(t, sql, "test_models.other.id")
}

func TestWithSelect_ExpressionPreserved(t *testing.T) {
	sql := buildSQL(t, WithSelect("count(*) as cnt"))
	assert.Contains(t, sql, "count(*)")
}

func TestWithSelect_Empty_SelectStar(t *testing.T) {
	sql := buildSQL(t, WithSelect())
	assert.Contains(t, sql, "*")
}

// ─── WithPagination ───────────────────────────────────────────────────────────

func TestWithPagination_LimitAndOffset(t *testing.T) {
	sql := buildSQL(t, WithPagination(Pagination{Limit: 10, Page: 3}))
	assert.Contains(t, sql, "LIMIT 10")
	assert.Contains(t, sql, "OFFSET 20")
}

func TestWithPagination_FirstPageNoOffset(t *testing.T) {
	sql := buildSQL(t, WithPagination(Pagination{Limit: 10, Page: 1}))
	assert.Contains(t, sql, "LIMIT 10")
	assert.NotContains(t, strings.ToUpper(sql), "OFFSET")
}

func TestWithPagination_ZeroLimitSkipped(t *testing.T) {
	sql := buildSQL(t, WithPagination(Pagination{Limit: 0, Page: 2}))
	assert.NotContains(t, strings.ToUpper(sql), "LIMIT")
}

// ─── WithLimit / WithOffset ───────────────────────────────────────────────────

func TestWithLimit_Applied(t *testing.T) {
	sql := buildSQL(t, WithLimit(5))
	assert.Contains(t, sql, "LIMIT 5")
}

func TestWithLimit_ZeroSkipped(t *testing.T) {
	sql := buildSQL(t, WithLimit(0))
	assert.NotContains(t, strings.ToUpper(sql), "LIMIT")
}

func TestWithOffset_Applied(t *testing.T) {
	sql := buildSQL(t, WithOffset(15))
	assert.Contains(t, sql, "OFFSET 15")
}

func TestWithOffset_ZeroSkipped(t *testing.T) {
	sql := buildSQL(t, WithOffset(0))
	assert.NotContains(t, strings.ToUpper(sql), "OFFSET")
}

// ─── WithScope ────────────────────────────────────────────────────────────────

func TestWithScope_AppliesCustomCondition(t *testing.T) {
	sql := buildSQL(t,
		WithScope(func(tx *gorm.DB) *gorm.DB {
			return tx.Where("views > ?", 18)
		}),
	)
	assert.Contains(t, sql, "views > ")
}

func TestWithScope_NilDoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		buildSQL(t, WithScope(nil))
	})
}

// ─── Композиция опций ─────────────────────────────────────────────────────────

func TestOptions_Composition(t *testing.T) {
	sql := buildSQL(t,
		WithConditions(map[string]any{"login": "john"}),
		WithOrder(Order{By: "login", Dir: OrderDirAsc}),
		WithPagination(Pagination{Limit: 20, Page: 2}),
		WithSelect("login", "email"),
	)
	assert.Contains(t, sql, "login")
	assert.Contains(t, strings.ToUpper(sql), "ORDER BY")
	assert.Contains(t, sql, "LIMIT 20")
	assert.Contains(t, sql, "OFFSET 20")
}

func TestOptions_CompositionMultipleConditionsAndOrder(t *testing.T) {
	sql := buildSQL(t,
		WithConditions(
			map[string]any{"login": "alice"},
			map[string]any{"login": "bob"},
		),
		WithOrder(Order{By: "views", Dir: OrderDirDesc}),
		WithLimit(5),
	)
	assert.Contains(t, strings.ToUpper(sql), " OR ")
	assert.Contains(t, sql, "views")
	assert.Contains(t, sql, "LIMIT 5")
}
