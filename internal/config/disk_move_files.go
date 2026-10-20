package config

import (
	"errors"
	"io/fs"
	"os"
)

type diskMoveFiles struct{}

func (diskMoveFiles) Exists(path string) bool {
	return IsFile(path)
}

func (diskMoveFiles) Read(path string) (string, error) {
	text, err := os.ReadFile(path)
	return string(text), err
}

func (diskMoveFiles) Write(path, text string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return err
	}
	_, err = file.WriteString(text)
	return errors.Join(err, file.Close())
}

func (diskMoveFiles) Rename(from, to string) error {
	return os.Rename(from, to)
}

func (diskMoveFiles) Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (diskMoveFiles) Folder(path string) error {
	return os.MkdirAll(path, 0o777)
}
