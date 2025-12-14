package file

import (
	"fmt"
	"os"
	"path/filepath"
)

func fileinfo(path string) (os.FileInfo, bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return stat, true, nil
	}
	if os.IsNotExist(err) {
		return stat, false, nil
	}
	return stat, false, err
}

func ResolveProjectPath(path string, name string) (string, error) {
	// -name flim -path ./bim/baz
	// ./bim does not exist:                         throw error
	// ./bim exists, is file:                        throw error
	// ./bim exists, ./bim/baz does not:             create @ ./bim/baz
	// ./bim/baz exists, is file:                    throw error
	// ./bim/baz exists, ./bim/baz/flim does not:    create @ ./bim/baz/flim
	// ./bim/baz/flim exists:                        throw error

	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working dir: %v", err)
		}
		path = cwd
	}

	dirName := filepath.Dir(path)
	dirStat, dirExists, err := fileinfo(dirName)
	if err != nil {
		return "", fmt.Errorf("failed to get file info for base directory %s: %v", dirName, err)
	}
	if !dirExists {
		return "", fmt.Errorf("base directory %s does not exist", dirName)
	}
	if !dirStat.IsDir() {
		return "", fmt.Errorf("base directory %s is a file", dirName)
	}

	pathStat, pathExists, err := fileinfo(path)
	if err != nil {
		return "", fmt.Errorf("failed to get file info for project path %s: %v", path, err)
	}
	if !pathExists {
		return path, nil
	}
	if !pathStat.IsDir() {
		return "", fmt.Errorf("project path %s is a file", path)
	}

	subDir := filepath.Join(path, name)
	_, subDirExists, err := fileinfo(subDir)
	if err != nil {
		return "", fmt.Errorf("failed to get file info for project path %s: %v", path, err)
	}
	if !subDirExists {
		return subDir, nil
	}
	return "", fmt.Errorf("project path %s already exists", subDir)
}
