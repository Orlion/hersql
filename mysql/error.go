package mysql

import "errors"

var (
	ErrBadConn       = errors.New("connection was bad")
	ErrMalformPacket = errors.New("malform packet error")

	ErrTxDone = errors.New("sql: Transaction has already been committed or rolled back")
)
