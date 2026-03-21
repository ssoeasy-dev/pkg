package repository

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RepositoryOption опция для настройки запросов
type RepositoryOption func(db *gorm.DB) *gorm.DB

// resolveTableName возвращает имя таблицы для квалификации полей.
// Сначала проверяет db.Statement.Table (заполняется после Parse или через Table()).
// Если пусто — берёт из интерфейса TableName() на модели (GORM-конвенция),
// иначе из NamingStrategy GORM. Это позволяет WithConditions работать
// даже когда Statement.Table ещё не заполнен до финализации запроса.
func resolveTableName(db *gorm.DB) string {
	if db.Statement.Table != "" {
		return db.Statement.Table
	}
	model := db.Statement.Model
	if model == nil {
		return ""
	}
	// Модель может реализовывать TableName() — тогда берём оттуда.
	type tableNamer interface{ TableName() string }
	if tn, ok := model.(tableNamer); ok {
		return tn.TableName()
	}
	// Fallback: GORM NamingStrategy из имени типа.
	t := reflect.TypeOf(model)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return ""
	}
	if db.Statement.DB != nil {
		return db.Statement.NamingStrategy.TableName(t.Name())
	}
	return ""
}

// normalizeForRawSQL конвертирует значения, которые pgx не умеет кодировать
// как параметры сырого SQL. Конкретный случай: uuid.UUID — это [16]byte (array),
// pgx не регистрирует для него pgtype-кодек, поэтому передача напрямую даёт
// "syntax error at or near $1". Если значение является array и реализует
// fmt.Stringer, возвращаем строковое представление.
func normalizeForRawSQL(v any) any {
	if v == nil {
		return nil
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Array {
		if stringer, ok := v.(fmt.Stringer); ok {
			return stringer.String()
		}
	}
	return v
}

// Like тип для LIKE-условий в WithConditions
type Like string

// IsNull тип для IS NULL / IS NOT NULL условий в WithConditions
type IsNull bool

// WithConditions добавляет условия WHERE.
//
// Несколько map объединяются через OR, поля внутри одной map — через AND:
//
//	WithConditions(m1)      → WHERE (m1.a AND m1.b)
//	WithConditions(m1, m2)  → WHERE (m1.a AND m1.b) OR (m2.a AND m2.b)
//
// Значение-слайс автоматически превращается в IN:
//
//	map[string]any{"id": []uuid.UUID{id1, id2}} → WHERE table.id IN (id1, id2)
//
// Для LIKE и IS NULL используйте типы-обёртки:
//
//	map[string]any{"login": Like("%john%")}
//	map[string]any{"deleted_at": IsNull(true)}
//
// ВАЖНО: не используйте операторы в ключах ("id IN", "created_at >") —
// квалификация таблицы сломает такие выражения. Для сложных условий
// используйте WithScope.
func WithConditions(conditions ...map[string]any) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		mainTable := resolveTableName(db)

		nonEmpty := make([]map[string]any, 0, len(conditions))
		for _, cond := range conditions {
			if len(cond) > 0 {
				nonEmpty = append(nonEmpty, cond)
			}
		}
		if len(nonEmpty) == 0 {
			return db
		}

		// qualify добавляет имя таблицы к полю если его нет.
		// Поля с точкой или пробелом не квалифицируются.
		qualify := func(field string) string {
			if mainTable == "" || strings.ContainsAny(field, ". ") {
				return field
			}
			return fmt.Sprintf("%s.%s", mainTable, field)
		}

		// buildGroup превращает одну map в SQL-фрагмент "(a = ? AND b LIKE ?)"
		// и список аргументов. Это позволяет корректно обрабатывать OR-группировку
		// без использования db.Or(*gorm.DB), который GORM интерпретирует как подзапрос.
		type sqlGroup struct {
			sql  string
			args []interface{}
		}

		buildGroup := func(cond map[string]any) sqlGroup {
			parts := make([]string, 0, len(cond))
			args := make([]interface{}, 0, len(cond))

			for field, value := range cond {
				key := qualify(field)
				switch v := value.(type) {
				case Like:
					parts = append(parts, key+" LIKE ?")
					args = append(args, string(v))
				case IsNull:
					if bool(v) {
						parts = append(parts, key+" IS NULL")
					} else {
						parts = append(parts, key+" IS NOT NULL")
					}
				default:
					rv := reflect.ValueOf(value)
					if rv.IsValid() && rv.Kind() == reflect.Slice {
						// Слайс → IN. GORM раскрывает "IN ?" в "(?,?,?)".
						parts = append(parts, key+" IN ?")
						args = append(args, value)
					} else {
						// Скаляр или фиксированный массив (например uuid.UUID = [16]byte).
						// normalizeForRawSQL конвертирует [N]byte в строку через Stringer,
						// потому что pgx не умеет кодировать array-типы как параметры raw SQL.
						parts = append(parts, key+" = ?")
						args = append(args, normalizeForRawSQL(value))
					}
				}
			}

			sql := strings.Join(parts, " AND ")
			if len(parts) > 1 {
				sql = "(" + sql + ")"
			}
			return sqlGroup{sql: sql, args: args}
		}

		if len(nonEmpty) == 1 {
			// Единственная группа — передаём напрямую
			g := buildGroup(nonEmpty[0])
			return db.Where(g.sql, g.args...)
		}

		// Несколько групп — объединяем через OR в одном Where.
		// Сырой SQL гарантирует корректную группировку скобками:
		//   (a=1 AND b=2) OR (a=3 AND b=4)
		sqls := make([]string, 0, len(nonEmpty))
		allArgs := make([]interface{}, 0)
		for _, cond := range nonEmpty {
			g := buildGroup(cond)
			sqls = append(sqls, g.sql)
			allArgs = append(allArgs, g.args...)
		}
		return db.Where(strings.Join(sqls, " OR "), allArgs...)
	}
}

// WithPreloads добавляет предзагрузку связей.
// Принимает опциональные условия для фильтрации preload-запроса.
func WithPreloads(preload string, opts ...RepositoryOption) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(opts) == 0 {
			return db.Preload(preload)
		}
		scope := func(db *gorm.DB) *gorm.DB {
			for _, opt := range opts {
				db = opt(db)
			}
			return db
		}
		return db.Preload(preload, scope)
	}
}

// JoinType тип JOIN'а
type JoinType string

const (
	JoinTypeInner JoinType = "JOIN"
	JoinTypeLeft  JoinType = "LEFT JOIN"
	JoinTypeRight JoinType = "RIGHT JOIN"
)

// JoinON описывает условие соединения.
// Поля без точки автоматически квалифицируются:
//
//	From: "user_id" → "main_table.user_id"
//	To:   "id"      → "join_table.id"
type JoinON struct {
	From string
	To   string
}

// Join описывает одно JOIN-соединение.
type Join struct {
	Type  JoinType // по умолчанию LEFT JOIN
	Table string
	On    JoinON
}

// WithJoins добавляет JOIN-соединения.
func WithJoins(joins ...Join) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		mainTable := resolveTableName(db)

		for _, join := range joins {
			if join.Table == "" || join.On.From == "" || join.On.To == "" {
				continue
			}

			joinType := join.Type
			if joinType == "" {
				joinType = JoinTypeLeft
			}

			from := join.On.From
			if !strings.Contains(from, ".") {
				from = fmt.Sprintf("%s.%s", mainTable, from)
			}

			to := join.On.To
			if !strings.Contains(to, ".") {
				to = fmt.Sprintf("%s.%s", join.Table, to)
			}

			expr := fmt.Sprintf("%s %s ON %s = %s", joinType, join.Table, from, to)
			db = db.Joins(expr)
		}
		return db
	}
}

// WithLimit добавляет лимит выборки.
func WithLimit(limit int) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			return db.Limit(limit)
		}
		return db
	}
}

// WithOffset добавляет смещение выборки.
func WithOffset(offset int) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if offset > 0 {
			return db.Offset(offset)
		}
		return db
	}
}

// OrderDir направление сортировки
type OrderDir string

const (
	OrderDirAsc  OrderDir = "asc"
	OrderDirDesc OrderDir = "desc"
)

// Order описывает одну сортировку.
// By передаётся as-is — используйте snake_case или полностью квалифицированное
// выражение ("users.created_at").
type Order struct {
	By  string
	Dir OrderDir
}

// WithOrder добавляет сортировку.
// Поля без точки и пробела квалифицируются именем основной таблицы.
func WithOrder(orders ...Order) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		mainTable := resolveTableName(db)

		for _, order := range orders {
			if order.By == "" {
				continue
			}

			dir := order.Dir
			if dir != OrderDirAsc && dir != OrderDirDesc {
				dir = OrderDirAsc
			}

			by := order.By
			if mainTable != "" && !strings.ContainsAny(by, ". ") {
				by = fmt.Sprintf("%s.%s", mainTable, by)
			}

			db = db.Order(by + " " + string(dir))
		}
		return db
	}
}

// WithSelect выбирает конкретные поля.
// Поля без точки, скобки или пробела квалифицируются именем основной таблицы.
func WithSelect(fields ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(fields) == 0 {
			return db
		}

		mainTable := resolveTableName(db)
		qualified := make([]string, len(fields))
		for i, field := range fields {
			if mainTable == "" || strings.ContainsAny(field, ". (") {
				qualified[i] = field
			} else {
				qualified[i] = fmt.Sprintf("%s.%s", mainTable, field)
			}
		}

		return db.Select(qualified)
	}
}

// WithScope добавляет произвольный GORM scope.
// Используйте для сложных условий: сырой SQL, подзапросы, составные выражения.
func WithScope(scope func(*gorm.DB) *gorm.DB) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if scope != nil {
			return scope(db)
		}
		return db
	}
}

// WithDeleted включает soft-deleted записи в выборку (Unscoped).
func WithDeleted(deleted bool) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if deleted {
			return db.Unscoped()
		}
		return db
	}
}

// Pagination параметры пагинации. Page начинается с 1.
type Pagination struct {
	Limit int
	Page  int
}

// WithPagination добавляет пагинацию.
func WithPagination(pagination Pagination) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if pagination.Limit <= 0 {
			return db
		}
		db = db.Limit(pagination.Limit)
		if pagination.Page > 1 {
			db = db.Offset((pagination.Page - 1) * pagination.Limit)
		}
		return db
	}
}

// WithClauses добавляет GORM clause-выражения.
// Пример: WithClauses(clause.OnConflict{DoNothing: true})
func WithClauses(clauses ...clause.Expression) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clauses...)
	}
}
