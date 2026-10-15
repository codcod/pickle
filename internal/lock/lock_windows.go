//go:build windows

package lock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryLock takes a non-blocking LockFileEx on the first byte of f (locking
// past EOF is legal, and the lock file stays empty). busy reports that
// another holder has it; err is any other failure.
func tryLock(f *os.File, exclusive bool) (busy bool, err error) {
	flags := uint32(windows.LOCKFILE_FAIL_IMMEDIATELY)
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	err = windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, new(windows.Overlapped))
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return true, nil
	}
	return false, err
}

func unlock(f *os.File) {
	_ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
