package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RepositoryOption опция для настройки запросов
type RepositoryOption func(db *gorm.DB) *gorm.DB

type Repository[Model any] interface {
	Create(ctx context.Context, value *Model, opts ...RepositoryOption) error
	Update(ctx context.Context, id uuid.UUID, value map[string]any, opts ...RepositoryOption) error
	Delete(ctx context.Context, id uuid.UUID, force bool, opts ...RepositoryOption) error
	FindOne(ctx context.Context, opts ...RepositoryOption) (*Model, error)
	FindAll(ctx context.Context, opts ...RepositoryOption) ([]Model, error)
	Count(ctx context.Context, opts ...RepositoryOption) (int64, error)
	Exists(ctx context.Context, opts ...RepositoryOption) (bool, error)

	DB(ctx context.Context) *gorm.DB
	
	// Базовые опции для построения запросов
	WithConditions(conditions ...interface{}) RepositoryOption
	WithPreloads(preloads ...string) RepositoryOption
	WithJoins(joins ...string) RepositoryOption
	WithLimit(limit int) RepositoryOption
	WithOffset(offset int) RepositoryOption
	WithOrder(order string) RepositoryOption
	WithSelect(fields ...string) RepositoryOption
	WithScope(scope func(*gorm.DB) *gorm.DB) RepositoryOption
	WithDeleted(deleted bool) RepositoryOption
}

type repository[Model any] struct {
	log        *logger.Logger
	txManager  tx.TxManager
	entityName string
}

func NewRepository[Model any](txManager tx.TxManager, log *logger.Logger, entityName string) Repository[Model] {
	return &repository[Model]{
		log:        log,
		txManager:  txManager,
		entityName: entityName,
	}
}

func (r *repository[Model]) DB(ctx context.Context) *gorm.DB {
	return r.txManager.GetDB(ctx)
}

// Create создает новую запись
func (r *repository[Model]) Create(ctx context.Context, value *Model, opts ...RepositoryOption) error {
	db := r.DB(ctx)

	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}
	
	err := db.Create(value).Error
	if err != nil {
		
		if strings.Contains(err.Error(), "23505") { // Unique violation
			return NewErrAlreadyExists(r.entityName)
		}
		if strings.Contains(err.Error(), "23503") { // Foreign key violation
			return NewErrForeignKeyViolation(r.entityName)
		}
		
		return NewErrCreateFailed(r.entityName)
	}
	return nil
}

// Update обновляет запись
func (r *repository[Model]) Update(ctx context.Context, ID uuid.UUID, updates map[string]any, opts ...RepositoryOption) error {
	db := r.DB(ctx)
	
	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}

	err := db.Model(new(Model)).Where("id = ?", ID).Updates(updates).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewErrNotFound(r.entityName)
		}
		return NewErrUpdateFailed(r.entityName)
	}
	return nil
}

// Delete удаляет запись
func (r *repository[Model]) Delete(ctx context.Context, id uuid.UUID, force bool, opts ...RepositoryOption) error {
	db := r.DB(ctx)
	
	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}
	
	if force {
		db = db.Unscoped()
	}

	var model Model
	err := db.Where("id = ?", id).Delete(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewErrNotFound(r.entityName)
		}
		return NewErrDeleteFailed(r.entityName)
	}
	return nil
}

// FindOne ищет одну запись
func (r *repository[Model]) FindOne(ctx context.Context, opts ...RepositoryOption) (*Model, error) {
	db := r.DB(ctx).Model(new(Model))
	
	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}
	
	var model Model
	err := db.First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErrNotFound(r.entityName)
		}
		return nil, NewErrGetFailed(r.entityName)
	}
	return &model, nil
}

// FindAll ищет все записи
func (r *repository[Model]) FindAll(ctx context.Context, opts ...RepositoryOption) ([]Model, error) {
	db := r.DB(ctx).Model(new(Model))
	
	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}
	
	var models []Model
	err := db.Find(&models).Error
	if err != nil {
		return nil, NewErrGetFailed(r.entityName)
	}
	return models, nil
}

// Count подсчитывает количество записей
func (r *repository[Model]) Count(ctx context.Context, opts ...RepositoryOption) (int64, error) {
	db := r.DB(ctx).Model(new(Model))
	
	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}
	
	var count int64
	err := db.Count(&count).Error
	if err != nil {
		return 0, NewErrGetFailed(r.entityName)
	}
	return count, nil
}

// Exists проверяет существование записей
func (r *repository[Model]) Exists(ctx context.Context, opts ...RepositoryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Опции для построения запросов

// WithConditions добавляет условия WHERE
func (r *repository[Model]) WithConditions(conditions ...interface{}) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(conditions) == 0 {
			return db
		}
		
		// Если передан map или структура
		if len(conditions) == 1 {
			cond := conditions[0]
			
			// Проверяем тип условий
			switch v := cond.(type) {
			case map[string]interface{}:
				return db.Where(v)
			case []interface{}:
				if len(v) > 0 {
					return db.Where(v[0], v[1:]...)
				}
			default:
				// Пробуем преобразовать структуру в условия
				condMap, err := structToMap(cond)
				if err == nil && len(condMap) > 0 {
					return db.Where(condMap)
				}
			}
		}
		
		// Для нескольких условий используем цепочку Where
		for i := 0; i < len(conditions); i += 2 {
			if i+1 < len(conditions) {
				db = db.Where(conditions[i], conditions[i+1])
			}
		}
		
		return db
	}
}

// WithPreloads добавляет предзагрузку связей
func (r *repository[Model]) WithPreloads(preloads ...string) RepositoryOption {
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
func (r *repository[Model]) WithJoins(joins ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		for _, join := range joins {
			db = db.Joins(join)
		}
		return db
	}
}

// WithLimit добавляет лимит
func (r *repository[Model]) WithLimit(limit int) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if limit > 0 {
			return db.Limit(limit)
		}
		return db
	}
}

// WithOffset добавляет смещение
func (r *repository[Model]) WithOffset(offset int) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if offset > 0 {
			return db.Offset(offset)
		}
		return db
	}
}

// WithOrder добавляет сортировку
func (r *repository[Model]) WithOrder(order string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if order != "" {
			return db.Order(order)
		}
		return db
	}
}

// WithSelect выбирает конкретные поля
func (r *repository[Model]) WithSelect(fields ...string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(fields) > 0 {
			return db.Select(fields)
		}
		return db
	}
}

// WithScope добавляет кастомный scope
func (r *repository[Model]) WithScope(scope func(*gorm.DB) *gorm.DB) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if scope != nil {
			return scope(db)
		}
		return db
	}
}

// WithDeleted включает удаленные записи
func (r *repository[Model]) WithDeleted(deleted bool) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if deleted {
			return db.Unscoped()
		}
		return db
	}
}

// Вспомогательные функции

// structToMap преобразует структуру в map для условий
func structToMap(obj interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	v := reflect.ValueOf(obj)
	// Если передан указатель, получаем значение
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	// Проверяем, что это структура
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", obj)
	}
	
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		// Пропускаем нулевые значения
		if !fieldValue.IsZero() {
			columnName := getColumnName(field)
			result[columnName] = fieldValue.Interface()
		}
	}
	
	return result, nil
}

// getFieldIndexByName ищет поле по имени
func getFieldIndexByName(t reflect.Type, name string) (int, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if strings.EqualFold(field.Name, name) {
			return i, true
		}
	}
	return -1, false
}

func getColumnName(field reflect.StructField) string {
	// Сначала проверяем тег gorm
	gormTag := field.Tag.Get("gorm")
	if gormTag != "" {
		if strings.Contains(gormTag, "column:") {
			re := regexp.MustCompile(`column:([^;]+)`)
			matches := re.FindStringSubmatch(gormTag)
			if len(matches) > 1 {
				return strings.TrimSpace(matches[1])
			}
		}
	}
	
	// Затем проверяем тег json
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}
	
	// Если нет тегов, преобразуем в snake_case
	return toSnakeCase(field.Name)
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Дополнительные функции для сложных запросов

// WithOR добавляет OR условия
func (r *repository[Model]) WithOR(conditions ...interface{}) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(conditions) == 0 {
			return db
		}
		
		query := db.Session(&gorm.Session{NewDB: true})
		for i := 0; i < len(conditions); i += 2 {
			if i+1 < len(conditions) {
				query = query.Or(conditions[i], conditions[i+1])
			}
		}
		
		return db.Where(query)
	}
}

// WithLike добавляет LIKE условия
func (r *repository[Model]) WithLike(field, value string) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		if value != "" {
			return db.Where(fmt.Sprintf("%s LIKE ?", field), "%"+value+"%")
		}
		return db
	}
}

// WithPagination добавляет пагинацию
func (r *repository[Model]) WithPagination(page, pageSize int) RepositoryOption {
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
func (r *repository[Model]) WithClauses(clauses ...clause.Expression) RepositoryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clauses...)
	}
}