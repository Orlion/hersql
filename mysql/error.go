package mysql

import (
	"errors"
	"fmt"
)

var (
	ErrBadConn       = errors.New("connection was bad")
	ErrMalformPacket = errors.New("malform packet error")

	ErrTxDone = errors.New("sql: Transaction has already been committed or rolled back")
)

type SqlError struct {
	Code    uint16
	Message string
	State   string
}

func (e *SqlError) Error() string {
	return fmt.Sprintf("ERROR %d (%s): %s", e.Code, e.State, e.Message)
}

func NewError(errCode uint16, message string) *SqlError {
	e := new(SqlError)
	e.Code = errCode

	if s, ok := MySQLState[errCode]; ok {
		e.State = s
	} else {
		e.State = DEFAULT_MYSQL_STATE
	}

	e.Message = message

	return e
}
