package utils

import "os"

// EnsureDirectory check the directory exists and create it if not exists
func EnsureDirectory(path string) error {
	_, err := os.Stat(path)
	if nil == err {
		return nil
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
	}
	return err
}
