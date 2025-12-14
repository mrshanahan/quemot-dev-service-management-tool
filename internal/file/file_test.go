package file

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func errIsNil(err error) error {
	if err != nil {
		return fmt.Errorf("expected error to be nil, but instead is '%v'", err)
	}
	return nil
}

func errMatches(patt string) func(err error) error {
	return func(err error) error {
		m, compileErr := regexp.MatchString(patt, err.Error())
		if compileErr != nil {
			return fmt.Errorf("invalid regex '%s': %v", patt, compileErr)
		}
		if !m {
			return fmt.Errorf("expected error to match pattern '%s', but instead is '%v'", patt, err)
		}
		return nil
	}
}

func pathIsEmpty(path string) error {
	if path != "" {
		return fmt.Errorf("expected path to be empty, but instead is '%s'", path)
	}
	return nil
}

func pathEndsWith(patt string) func(path string) error {
	return func(path string) error {
		if !strings.HasSuffix(path, patt) {
			return fmt.Errorf("expected path to end with '%s', but instead is '%s'", patt, path)
		}
		return nil
	}
}

func TestResolveProjectPath(t *testing.T) {
	var tests = []struct {
		testName                string
		path                    string
		name                    string
		existingEntries         []string
		expectedProjectPathPred func(string) error
		expectedErrorPred       func(error) error
	}{
		{
			"base dir doesn't exist",
			"bim/baz",
			"flim",
			[]string{},
			pathIsEmpty,
			errMatches("base directory .*/bim does not exist"),
		},
		{
			"base dir is file",
			"bim/baz",
			"flim",
			[]string{"f:bim"},
			pathIsEmpty,
			errMatches("base directory .*/bim is a file"),
		},
		{
			"base dir exists, dir does not",
			"bim/baz",
			"flim",
			[]string{"d:bim"},
			pathEndsWith("/bim/baz"),
			errIsNil,
		},
		{
			"path is file",
			"bim/baz",
			"flim",
			[]string{"d:bim", "f:bim/baz"},
			pathIsEmpty,
			errMatches("project path .*/bim/baz is a file"),
		},
		{
			"path is dir, name subdir does not exist",
			"bim/baz",
			"flim",
			[]string{"d:bim/baz"},
			pathEndsWith("/bim/baz/flim"),
			errIsNil,
		},
		{
			"path is dir, name subdir exists",
			"bim/baz",
			"flim",
			[]string{"d:bim/baz/flim"},
			pathIsEmpty,
			errMatches("project path .*/bim/baz/flim already exists"),
		},
		{
			"path is dir, name subdir is file",
			"bim/baz",
			"flim",
			[]string{"d:bim/baz", "f:bim/baz/flim"},
			pathIsEmpty,
			errMatches("project path .*/bim/baz/flim already exists"),
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(s *testing.T) {
			dir, err := os.MkdirTemp("", "file-test-*")
			if err != nil {
				s.Fatalf("unable to create temp dir: %v", err)
			}
			defer os.RemoveAll(dir)

			for _, e := range test.existingEntries {
				comps := strings.Split(e, ":")
				if len(comps) != 2 {
					s.Fatalf("invalid entry, expecting format '(f|d):<path>', got '%s'", e)
				}
				switch comps[0] {
				case "f":
					_, err := os.Create(filepath.Join(dir, comps[1]))
					if err != nil {
						s.Fatalf("unable to create existing file entry '%s': %v", comps[1], err)
					}
				case "d":
					err := os.MkdirAll(filepath.Join(dir, comps[1]), 0700)
					if err != nil {
						s.Fatalf("unable to create existing dir entry '%s': %v", comps[1], err)
					}
				default:
					s.Fatalf("unrecognized existing entry option: %s", comps[0])
				}
			}

			path := filepath.Join(dir, test.path)

			actualProjectPath, actualError := ResolveProjectPath(path, test.name)
			if err := test.expectedProjectPathPred(actualProjectPath); err != nil {
				s.Errorf("%v", err)
			}
			if err := test.expectedErrorPred(actualError); err != nil {
				s.Errorf("%v", err)
			}
		})
	}
}
