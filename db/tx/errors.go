package tx

import "errors"

var (
	ErrTxBegin    = errors.New("failed to begin transaction")
	ErrTxCommit   = errors.New("failed to commit transaction")
	ErrTxRollback = errors.New("failed to rollback transaction")
)
