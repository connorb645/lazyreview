package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

// SaveListChange changes one list setting in the config file at path, as
// ListEdited does, and writes it whole: to a new file in the same folder,
// renamed over the old one, so a failure leaves the old file as it was. A
// path that is a link keeps the link, and the file it points to changes,
// keeping its mode. With no file at path, it makes one, and its folder,
// holding only the list. A change that changes nothing writes nothing. Its
// error names the file, and why.
func SaveListChange(tr *i18n.TranslationSet, path, key string, change ListChange) error {
	saved := func() error {
		target, mode := path, os.FileMode(0o644)
		text, exists, err := utils.ReadText(path)
		if err != nil {
			return err
		}
		if exists {
			if target, err = filepath.EvalSymlinks(path); err != nil {
				return err
			}
			info, err := os.Stat(target)
			if err != nil {
				return err
			}
			mode = info.Mode().Perm()
		}
		edited, err := ListEdited(tr, text, key, change)
		if err != nil || edited == text {
			return err
		}
		if !exists {
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
		}
		return utils.WriteWhole(target, edited, mode)
	}()
	if saved != nil {
		return errors.New(utils.ResolvePlaceholderString(tr.ConfigEditFailed, map[string]string{"path": path, "why": saved.Error()}))
	}
	return nil
}
