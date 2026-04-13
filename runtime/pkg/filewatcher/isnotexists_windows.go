//go:build windows

package filewatcher

import (
	"os"
)

func isNotExists(err error) bool {
	return os.IsNotExist(err)
}
