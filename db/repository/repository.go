package repository

import (
	"context"

	"github.com/ssoeasy-dev/pkg/db"
	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/errors"
	"github.com/ssoeasy-dev/pkg/logger"

	"gorm.io/gorm"
)

type Repository[Model any] interface {
	// DB возвращает *gorm.DB привязанный к текущей транзакции (если есть).
	// Используйте как аргумент для пакетных хелперов Query[T] / QueryOne[T]
	// когда нужно сканировать результат в тип отличный от Model.
	DB(ctx context.Context) *gorm.DB

	Create(ctx context.Context, value *Model, opts ...RepositoryOption) error

	// Update обновляет записи. value — map или указатель на struct.
	// Фильтрация — через WithConditions / WithScope в opts.
	// Возвращает количество затронутых строк; 0 строк — не ошибка.
	Update(ctx context.Context, value map[string]any, opts ...RepositoryOption) (int64, error)

	// Delete удаляет записи по условиям из opts.
	// force=true — hard delete (Unscoped), false — soft delete через deleted_at.
	// Возвращает количество затронутых строк; 0 строк — не ошибка.
	Delete(ctx context.Context, force bool, opts ...RepositoryOption) (int64, error)

	// FindOne возвращает первую запись удовлетворяющую условиям.
	// Порядок определяется WithOrder; без него порядок не гарантирован.
	// Возвращает ErrNotFound если запись не найдена.
	FindOne(ctx context.Context, opts ...RepositoryOption) (*Model, error)

	FindAll(ctx context.Context, opts ...RepositoryOption) ([]Model, error)
	Count(ctx context.Context, opts ...RepositoryOption) (int64, error)
	Exists(ctx context.Context, opts ...RepositoryOption) (bool, error)
}

type repository[Model any] struct {
	log        logger.Logger
	txManager  tx.TxManager
	EntityName string
}

func NewRepository[Model any](txManager tx.TxManager, log logger.Logger, entityName string) Repository[Model] {
	return &repository[Model]{
		log:        log,
		txManager:  txManager,
		EntityName: entityName,
	}
}

func (r *repository[Model]) DB(ctx context.Context) *gorm.DB {
	// Session(NewDB: true) — свежий клон, не делит Statement с другими репозиториями.
	// НЕ вызываем Statement.Parse() явно: он нарушает построение Statement в некоторых
	// версиях GORM когда Find/Find(&[]Model{}) вызывается после него.
	// WithConditions получает имя таблицы через Statement.Table (если заполнен)
	// или через TableName() модели напрямую.
	return r.txManager.GetDB(ctx).Session(&gorm.Session{NewDB: true}).Model(new(Model))
}

func (r *repository[Model]) Create(ctx context.Context, value *Model, opts ...RepositoryOption) error {
	if value == nil {
		return errors.Newf(errors.ErrInvalidArgument, "%s not provided", r.EntityName)
	}
	DB := r.DB(ctx)
	for _, opt := range opts {
		DB = opt(DB)
	}
	if err := DB.Create(value).Error; err != nil {
		return db.NewError(err, r.EntityName)
	}
	return nil
}

func (r *repository[Model]) Update(ctx context.Context, value map[string]any, opts ...RepositoryOption) (int64, error) {
	DB := r.DB(ctx)
	for _, opt := range opts {
		DB = opt(DB)
	}
	result := DB.Updates(value)
	if result.Error != nil {
		return 0, db.NewError(result.Error, r.EntityName)
	}
	return result.RowsAffected, nil
}

func (r *repository[Model]) Delete(ctx context.Context, force bool, opts ...RepositoryOption) (int64, error) {
	DB := r.DB(ctx)
	for _, opt := range opts {
		DB = opt(DB)
	}
	if force {
		DB = DB.Unscoped()
	}
	result := DB.Delete(new(Model))
	if result.Error != nil {
		return 0, db.NewError(result.Error, r.EntityName)
	}
	return result.RowsAffected, nil
}

func (r *repository[Model]) FindOne(ctx context.Context, opts ...RepositoryOption) (*Model, error) {
	DB := r.DB(ctx)
	for _, opt := range opts {
		DB = opt(DB)
	}
	var model Model
	// Take вместо First: не добавляет неявный ORDER BY primary_key.
	if err := DB.Take(&model).Error; err != nil {
		return nil, db.NewError(err, r.EntityName)
	}
	return &model, nil
}

func (r *repository[Model]) FindAll(ctx context.Context, opts ...RepositoryOption) ([]Model, error) {
	DB := r.DB(ctx)
	for _, opt := range opts {
		DB = opt(DB)
	}
	var models []Model
	if err := DB.Find(&models).Error; err != nil {
		return nil, db.NewError(err, r.EntityName)
	}
	return models, nil
}

func (r *repository[Model]) Count(ctx context.Context, opts ...RepositoryOption) (int64, error) {
	DB := r.DB(ctx)
	for _, opt := range opts {
		DB = opt(DB)
	}
	var count int64
	if err := DB.Count(&count).Error; err != nil {
		return 0, db.NewError(err, r.EntityName)
	}
	return count, nil
}

func (r *repository[Model]) Exists(ctx context.Context, opts ...RepositoryOption) (bool, error) {
	count, err := r.Count(ctx, opts...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
