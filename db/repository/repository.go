package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/ssoeasy-dev/pkg/db/tx"
	"github.com/ssoeasy-dev/pkg/logger"
	"gorm.io/gorm"
)

type Repository[Model any] interface {
	Create(ctx context.Context, value *Model) error
	Update(ctx context.Context, value *Model) error
	Delete(ctx context.Context, id uuid.UUID, force bool) error
	GetByID(ctx context.Context, id uuid.UUID) (*Model, error)
	FindOne(ctx context.Context, conditions map[string]any) (*Model, error)
	FindAll(ctx context.Context, conditions map[string]any, limit, offset int) ([]Model, error)
	Count(ctx context.Context, conditions map[string]any) (int64, error)
	Exists(ctx context.Context, conditions map[string]any) (bool, error)
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

func (r *repository[Model]) getDB(ctx context.Context) *gorm.DB {
	return r.txManager.GetDB(ctx)
}

func (r *repository[Model]) Create(ctx context.Context, value *Model) error {
	err := r.getDB(ctx).Create(value).Error
	if err != nil {
		r.log.Debug(ctx, "Create error", map[string]any{
			"entity": r.entityName,
			"error": err.Error(),
		})
		if strings.Contains(err.Error(), "23505") {
			return NewErrAlreadyExists(r.entityName)
		}
		return NewErrCreateFailed(r.entityName)
	}
	return nil
}

func (r *repository[Model]) Update(ctx context.Context, value *Model) error {
	err := r.getDB(ctx).Save(value).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewErrNotFound(r.entityName)
		}
		return NewErrUpdateFailed(r.entityName)
	}
	return nil
}

func (r *repository[Model]) Delete(ctx context.Context, id uuid.UUID, force bool) error {
	query := r.getDB(ctx)
	if force {
		query = query.Unscoped()
	}

	var model Model
	err := query.Where("id = ?", id).Delete(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewErrNotFound(r.entityName)
		}
		return NewErrDeleteFailed(r.entityName)
	}
	return nil
}

func (r *repository[Model]) GetByID(ctx context.Context, id uuid.UUID) (*Model, error) {
	var model Model
	err := r.getDB(ctx).
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErrNotFound(r.entityName)
		}
		return nil, NewErrGetFailed(r.entityName)
	}

	return &model, nil
}

func (r *repository[Model]) FindOne(ctx context.Context, conditions map[string]interface{}) (*Model, error) {
	var model Model
	query := r.getDB(ctx)

	for key, value := range conditions {
		query = query.Where(key, value)
	}

	err := query.First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewErrNotFound(r.entityName)
		}
		return nil, NewErrGetFailed(r.entityName)
	}
	return &model, nil
}

func (r *repository[Model]) FindAll(ctx context.Context, conditions map[string]interface{}, limit, offset int) ([]Model, error) {
	var models []Model
	query := r.getDB(ctx)

	for key, value := range conditions {
		query = query.Where(key, value)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&models).Error
	if err != nil {
		return nil, NewErrGetFailed(r.entityName)
	}
	return models, nil
}

func (r *repository[Model]) Count(ctx context.Context, conditions map[string]interface{}) (int64, error) {
	var count int64
	query := r.getDB(ctx).Model(new(Model))

	for key, value := range conditions {
		query = query.Where(key, value)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, NewErrGetFailed(r.entityName)
	}
	return count, nil
}

func (r *repository[Model]) Exists(ctx context.Context, conditions map[string]any) (bool, error) {
	count, err := r.Count(ctx, conditions)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
