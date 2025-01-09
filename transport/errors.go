package transport

import (
	"errors"
	"fmt"
)

var (
	ErrMalformPkt            = errors.New("malformed packet")
	ErrUnsupportedAuthPlugin = errors.New("unsupported authentication plugin")
)

func newErrUnsupportedAuthPlugin(plugin string) error {
	return fmt.Errorf("%w: %s, please use mysql_native_password or caching_sha2_password", ErrUnsupportedAuthPlugin, plugin)
}
