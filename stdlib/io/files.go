package io

import "os"

// ReadText reads the entire file at path as a string. It replaces the
// deprecated io/ioutil calls with os equivalents (Go 1.16+).
func ReadText(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

// WriteText writes data to path, creating or truncating the file (0644).
func WriteText(path, data string) error {
	return os.WriteFile(path, []byte(data), 0o644)
}

// ReadBytes reads the entire file at path as bytes.
func ReadBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteBytes writes raw bytes to path (0644).
func WriteBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// ReadFile is an alias for ReadText, matching the name used in Rayo examples
// (io.ReadFile).
func ReadFile(path string) (string, error) {
	return ReadText(path)
}

// WriteFile is an alias for WriteText.
func WriteFile(path, data string) error {
	return WriteText(path, data)
}

// Exists reports whether a file or directory exists at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
