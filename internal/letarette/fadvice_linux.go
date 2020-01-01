package letarette

import (
	"golang.org/x/sys/unix"
)

func fadvice(fd uintptr, size int64) error {
	return unix.Fadvise(int(fd), 0, size, unix.FADV_RANDOM)
}
