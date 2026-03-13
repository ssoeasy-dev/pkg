package repository

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RepositoryOption опция для настройки запросов
type RepositoryOption func(db *gorm.DB) *gorm.DB

// Like тип для LIKE-условий в WithConditions
type Like string
// IsNull тип для IsNull-условий в WithConditions
type IsNull bool

// WithConditions добавляет условия WHERE
// Несколько map объединяются через OR, поля внутри одной map — через AND
func WithConditions(conditions ...map[string]any) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		mainTable := db.Statement.Table

		nonEmpty := make([]map[string]any, 0, len(conditions))
		for _, cond := range conditions {
			if len(cond) > 0 {
				nonEmpty = append(nonEmpty, cond)
			}
		}
		if len(nonEmpty) == 0 {
			return db
		}

		qualify := func(field string) string {
			if strings.Contains(field, ".") {
				return field
			}
			return fmt.Sprintf("%s.%s", mainTable, field)
		}

		applyCondition := func(db *gorm.DB, cond map[string]any, or bool) *gorm.DB {
			for field, value := range cond {
				apply := db.Where
				if or {
					apply = db.Or
				}
				if like, ok := value.(Like); ok {
					db = apply(qualify(field)+" LIKE ?", string(like))
				} else if isNull, ok := value.(IsNull); ok {
					opp := ""
					if !isNull {
						opp = " NOT"
					}
					db = apply(qualify(field)+" IS" + opp + " NULL")
				} else {
					db = apply(map[string]any{qualify(field): value})
				}
			}
			return db
		}

		db = applyCondition(db, nonEmpty[0], false)
		for _, cond := range nonEmpty[1:] {
			db = applyCondition(db, cond, true)
		}

		return db
	}
}

// WithPreloads добавляет предзагрузку связей
func WithPreloads(preloads ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		for _, preload := range preloads {
			db = db.Preload(preload)
		}
		return db
	}
}

// JoinType тип JOIN'а
type JoinType string

const (
	JoinTypeInner JoinType = "JOIN"
	JoinTypeLeft  JoinType = "LEFT JOIN"
	JoinTypeRight JoinType = "RIGHT JOIN"
)

type JoinON struct {
	From string
	To   string
}

type Join struct {
	Type  JoinType
	Table string
	On    JoinON
}

// WithJoins добавляет JOIN'ы
func WithJoins(joins ...Join) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		mainTable := db.Statement.Table

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

			clause := fmt.Sprintf("%s %s ON %s = %s", joinType, join.Table, from, to)
			db = db.Joins(clause)
		}
		return db
	}
}

// WithLimit добавляет лимит
func WithLimit(limit int) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			return db.Limit(limit)
		}
		return db
	}
}

// WithOffset добавляет смещение
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

type Order struct {
	By  string
	Dir OrderDir
}

// WithOrder добавляет сортировку
func WithOrder(orders ...Order) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		mainTable := db.Statement.Table

		for _, order := range orders {
			if order.By == "" {
				continue
			}

			dir := order.Dir
			if dir != OrderDirAsc && dir != OrderDirDesc {
				dir = OrderDirAsc
			}

			by := toSnakeCase(order.By)
			if !strings.Contains(by, ".") {
				by = fmt.Sprintf("%s.%s", mainTable, by)
			}

			db = db.Order(by + " " + string(dir))
		}
		return db
	}
}

// WithSelect выбирает конкретные поля
func WithSelect(fields ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(fields) == 0 {
			return db
		}

		mainTable := db.Statement.Table
		qualified := make([]string, len(fields))
		for i, field := range fields {
			if strings.Contains(field, ".") {
				qualified[i] = field
			} else {
				qualified[i] = fmt.Sprintf("%s.%s", mainTable, field)
			}
		}

		return db.Select(qualified)
	}
}

// WithScope добавляет кастомный scope
func WithScope(scope func(*gorm.DB) *gorm.DB) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if scope != nil {
			return scope(db)
		}
		return db
	}
}

// WithDeleted включает удаленные записи
func WithDeleted(deleted bool) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if deleted {
			return db.Unscoped()
		}
		return db
	}
}

// Pagination параметры пагинации
type Pagination struct {
	Limit int
	Page  int
}

// WithPagination добавляет пагинацию
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

// WithClauses добавляет GORM clauses
func WithClauses(clauses ...clause.Expression) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clauses...)
	}
}
