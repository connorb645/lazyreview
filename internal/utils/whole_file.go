package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"strconv"
)

// WriteWhole writes text to a new file beside path, made with mode less the
// umask, then renames it over path: another process reading it finds the
// file as it was or as it is now, never empty or half written. A failure
// leaves path as it was.
func WriteWhole(path, text string, mode fs.FileMode) error {
	tail := make([]byte, 4)
	if _, err := rand.Read(tail); err != nil {
		return err
	}
	partial := path + "." + strconv.Itoa(os.Getpid()) + "." + hex.EncodeToString(tail) + ".tmp"
	if err := os.WriteFile(partial, []byte(text), mode); err != nil {
		_ = os.Remove(partial)
		return err
	}
	if err := os.Rename(partial, path); err != nil {
		_ = os.Remove(partial)
		return err
	}
	return nil
}

// ReadText is a file's text, and false when there is no such file.
func ReadText(path string) (string, bool, error) {
	text, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(text), true, nil
}
