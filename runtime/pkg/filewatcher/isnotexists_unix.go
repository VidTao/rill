//go:build !windows

package filewatcher

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func isNotExists(err error) bool {
	return os.IsNotExist(err) || errors.Is(err, unix.ENOENT)
}
