package repository

import "fmt"

func NewErrNotFound(entity string) error {
	return fmt.Errorf("%s: not found", entity)
}

func NewErrAlreadyExists(entity string) error {
	return fmt.Errorf("%s: already exists", entity)
}

func NewErrForeignKeyViolation(entity string) error {
	return fmt.Errorf("%s: foreign key violation", entity)
}

func NewErrNothingToUpdate(entity string) error {
	return fmt.Errorf("%s: nothing to update", entity)
}

func NewErrCreateFailed(entity string) error {
	return fmt.Errorf("%s: create failed", entity)
}

func NewErrUpdateFailed(entity string) error {
	return fmt.Errorf("%s: update failed", entity)
}

func NewErrDeleteFailed(entity string) error {
	return fmt.Errorf("%s: delete failed", entity)
}

func NewErrGetFailed(entity string) error {
	return fmt.Errorf("%s: get failed", entity)
}
