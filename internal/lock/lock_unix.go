//go:build unix

package lock

import (
	"os"
	"syscall"
)

// tryLock takes a non-blocking flock on f. busy reports that another holder
// has it; err is any other failure.
func tryLock(f *os.File, exclusive bool) (busy bool, err error) {
	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	err = syscall.Flock(int(f.Fd()), how|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return true, nil
	}
	return false, err
}

func unlock(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
