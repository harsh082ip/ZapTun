package config

import "os"

// fileExists checks if a file exists at the given path.
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	// os.IsNotExist returns true if the error is that the file does not exist.
	return !os.IsNotExist(err)
}
