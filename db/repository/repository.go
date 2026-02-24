package repository

import (
	"context"

	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/logger"
	"gorm.io/gorm"
)

type Repository[Model any] interface {
	DB(ctx context.Context) *gorm.DB
	Create(ctx context.Context, value *Model, opts ...RepositoryOption) error
	Update(ctx context.Context, value map[string]any, opts ...RepositoryOption) (int64, error)
	Delete(ctx context.Context, force bool, opts ...RepositoryOption) (int64, error)
	FindOne(ctx context.Context, opts ...RepositoryOption) (*Model, error)
	FindAll(ctx context.Context, opts ...RepositoryOption) ([]Model, error)
	Count(ctx context.Context, opts ...RepositoryOption) (int64, error)
	Exists(ctx context.Context, opts ...RepositoryOption) (bool, error)
}

type repository[Model any] struct {
	log        *logger.Logger
	txManager  tx.TxManager
	EntityName string
}

func NewRepository[Model any](txManager tx.TxManager, log *logger.Logger, EntityName string) Repository[Model] {
	return &repository[Model]{
		log:        log,
		txManager:  txManager,
		EntityName: EntityName,
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
		return NewRepositoryError(err, r.EntityName, ErrCreationFailed)
	}
	return nil
}

// Update обновляет запись
func (r *repository[Model]) Update(ctx context.Context, updates map[string]any, opts ...RepositoryOption) (int64, error) {
	db := r.DB(ctx)
	
	// Применяем опции
	for _, opt := range opts {
		db = opt(db)
	}

	result := db.Model(new(Model)).Updates(updates)
	if result.Error != nil {
		return 0, NewRepositoryError(result.Error, r.EntityName, ErrUpdateFailed)
	}
	if result.RowsAffected == 0 {
		return 0, NewRepositoryError(gorm.ErrRecordNotFound, r.EntityName, ErrNotFound)
	}
	return result.RowsAffected, nil
}

// Delete удаляет запись
func (r *repository[Model]) Delete(ctx context.Context, force bool, opts ...RepositoryOption) (int64, error) {
    db := r.DB(ctx)

    for _, opt := range opts {
        db = opt(db)
    }

    if force {
        db = db.Unscoped()
    }

    result := db.Delete(new(Model))
    if result.Error != nil {
        return 0, NewRepositoryError(result.Error, r.EntityName, ErrDeleteFailed)
    }
    if result.RowsAffected == 0 {
        return 0, NewRepositoryError(gorm.ErrRecordNotFound, r.EntityName, ErrNotFound)
    }
    return result.RowsAffected, nil
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
		return nil, NewRepositoryError(err, r.EntityName, ErrGetFailed)
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
		return nil, NewRepositoryError(err, r.EntityName, ErrGetFailed)
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
		return 0, NewRepositoryError(err, r.EntityName, ErrGetFailed)
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

// RawQuery выполняет голый запрос
func (r *repository[Model]) RawQuery(ctx context.Context, sql string, args []any, res []any) ([]Model, error) {
    var results []Model
    err := r.DB(ctx).Raw(sql, args...).Scan(&results).Error
    return results, err
}
