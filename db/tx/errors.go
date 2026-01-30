package tx

import (
	"errors"

	"github.com/ssoeasy-dev/pkg/db"
)

var (
	ErrTxBegin    db.DBError = errors.New("failed to begin transaction")
	ErrTxCommit   db.DBError = errors.New("failed to commit transaction")
	ErrTxRollback db.DBError = errors.New("failed to rollback transaction")
)