package file

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

func CopyFSWithReplacment(
	fsys fs.FS,
	src string,
	dst string,
	filter func(d fs.DirEntry, path string) bool,
	nameHandler func(d fs.DirEntry, path string) string,
	dataHandler func(d fs.DirEntry, data []byte) []byte) error {
	return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relSrcPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if !filter(d, relSrcPath) {
			return nil
		}

		relDstPath := nameHandler(d, relSrcPath)
		dstPath := filepath.Join(dst, relDstPath)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o700)
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		data = dataHandler(d, data)
		return os.WriteFile(dstPath, data, 0o644)
	})

}

func IsNotIgnored(d fs.DirEntry, path string) bool {
	return !strings.HasPrefix(d.Name(), "IGNORE__")
}

var (
	VariablePattern *regexp.Regexp = regexp.MustCompile(`{{([a-zA-Z0-9_]+)}}`)
)

func HandleNameFactory(variables map[string]string) func(d fs.DirEntry, path string) string {
	return func(d fs.DirEntry, path string) string {
		renamedBytes := VariablePattern.ReplaceAllFunc([]byte(path), func(bs []byte) []byte {
			val, prs := variables[string(bs[2:len(bs)-2])]
			if prs {
				return []byte(val)
			}
			return bs
		})
		return string(renamedBytes)
	}
}

func HandleDataFactory(variables map[string]string) func(d fs.DirEntry, data []byte) []byte {
	return func(d fs.DirEntry, data []byte) []byte {
		replacedBytes := VariablePattern.ReplaceAllFunc(data, func(bs []byte) []byte {
			val, prs := variables[string(bs[2:len(bs)-2])]
			if prs {
				return []byte(val)
			}
			return bs
		})
		return replacedBytes
	}
}

func CopyTemplate(fsys fs.FS, src string, dst string, variables map[string]string) error {
	return CopyFSWithReplacment(fsys, src, dst, IsNotIgnored, HandleNameFactory(variables), HandleDataFactory(variables))
}
