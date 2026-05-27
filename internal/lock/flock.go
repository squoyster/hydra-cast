package lock

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
)

type FileLock struct {
	path string
	fd   int
}

func New(path string) *FileLock {
	return &FileLock{path: path}
}

func (l *FileLock) TryLock() error {
	fd, err := syscall.Open(l.path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		syscall.Close(fd)

		if err == syscall.EWOULDBLOCK {
			if stale, _ := l.isStale(); stale {
				if removeErr := os.Remove(l.path); removeErr != nil {
					return fmt.Errorf("remove stale lock: %w", removeErr)
				}
				return l.TryLock()
			}
			return fmt.Errorf("another instance is running")
		}
		return fmt.Errorf("acquire lock: %w", err)
	}

	l.fd = fd

	if _, err := syscall.Write(fd, []byte(strconv.Itoa(os.Getpid()))); err != nil {
		syscall.Close(fd)
		return fmt.Errorf("write pid to lock file: %w", err)
	}

	return nil
}

func (l *FileLock) isStale() (bool, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return true, nil
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return true, nil
	}

	procPath := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(procPath); os.IsNotExist(err) {
		return true, nil
	}

	return false, nil
}

func (l *FileLock) Unlock() error {
	if l.fd == 0 {
		return nil
	}

	if err := syscall.Flock(l.fd, syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock: %w", err)
	}

	syscall.Close(l.fd)
	l.fd = 0

	os.Remove(l.path)

	return nil
}

func (l *FileLock) Path() string {
	return l.path
}

const StaleThreshold = 1 * time.Hour

func init() {
	_ = StaleThreshold
}
