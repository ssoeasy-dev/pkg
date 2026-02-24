package repository

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RepositoryOption опция для настройки запросов
type RepositoryOption func(db *gorm.DB) *gorm.DB

// WithConditions добавляет условия WHERE
func WithConditions(conditions ...map[string]any) RepositoryOption {
    return func(db *gorm.DB) *gorm.DB {
        // Отфильтровываем пустые условия, чтобы не создавать лишних конструкций
        nonEmpty := make([]map[string]any, 0, len(conditions))
        for _, cond := range conditions {
            if len(cond) > 0 {
                nonEmpty = append(nonEmpty, cond)
            }
        }
        if len(nonEmpty) == 0 {
            return db
        }

        // Первое условие добавляем через Where (оно становится базовым)
        db = db.Where(nonEmpty[0])
        // Остальные добавляем через Or, что создаёт правильную группировку
        for _, cond := range nonEmpty[1:] {
            db = db.Or(cond)
        }
        return db
    }
}

// WithPreloads добавляет предзагрузку связей
func WithPreloads(preloads ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		for _, preload := range preloads {
			if strings.Contains(preload, ".") {
				// Для вложенных прелоадов
				db = db.Preload(preload)
			} else {
				// Для простых прелоадов
				db = db.Preload(preload)
			}
		}
		return db
	}
}

// WithJoins добавляет JOIN'ы
func WithJoins(joins ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		for _, join := range joins {
			db = db.Joins(join)
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

// WithOrder добавляет сортировку
func WithOrder(order string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if order != "" {
			return db.Order(order)
		}
		return db
	}
}

// WithSelect выбирает конкретные поля
func WithSelect(fields ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(fields) > 0 {
			return db.Select(fields)
		}
		return db
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

// WithPagination добавляет пагинацию
func WithPagination(page, pageSize int) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if pageSize > 0 {
			db = db.Limit(pageSize)
			if page > 1 {
				db = db.Offset((page - 1) * pageSize)
			}
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
