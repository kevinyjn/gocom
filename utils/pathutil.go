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

// IsPathExists check if the path exists
func IsPathExists(path string) bool {
	_, err := os.Stat(path)
	if nil == err {
		return true
	}
	return os.IsExist(err)
}

// IsPathNotExists check if the path not exists
func IsPathNotExists(path string) bool {
	_, err := os.Stat(path)
	if nil == err {
		return false
	}
	return os.IsNotExist(err)
}
