package utils

import (
	"errors"
	"os"
)

func DirExists(path string) (bool, error) {
	finfo, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return finfo.IsDir(), nil
}
