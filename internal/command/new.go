package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/file"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

type ProjectType int

const (
	SERVICE ProjectType = iota
)

var (
	SUPPORTED_PROJECT_TYPES map[string]ProjectType = map[string]ProjectType{
		"service": SERVICE,
	}

	SUPPORTED_PROJECT_TYPES_STR string = strings.Join(utils.Keys(SUPPORTED_PROJECT_TYPES), ", ")
)

type NewCommandSpec struct {
	Args []string
}

func (s *NewCommandSpec) Usage() string {
	return fmt.Sprintf(`Usage:
    smt new <options>

    Options:
        -path <string>  Path to the new project
                        - If the path does not exist, the project will be created there (irrespective of -name).
                        - If the path is a directory, the project will be created as a subdirectory therein with the name from -name.
                        - All other scenarios result in an error.
    
        -name <string>  Name of the project
                        This will be used as the repository name where possible (see -path) and executable name where appropriate.
    
        -type <string>  Type of project to create.
                        Available types: %s
`, SUPPORTED_PROJECT_TYPES_STR)
}

func (s *NewCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})
	nameParam := fs.String("name", "", "Name of the service")
	pathParam := fs.String("path", "", "Path where the base directory should be placed.\nE.g. \"foo/bar\" would indicate that the service \"bip\" would be at \"foo/bar/bip\".")
	typeParam := fs.String("type", "service", fmt.Sprintf("Type of project to create. Available types: %s", SUPPORTED_PROJECT_TYPES_STR))

	if err := fs.Parse(s.Args); err != nil {
		if err == flag.ErrHelp {
			return nil, err
		}
		return nil, fmt.Errorf("failed to parse command line: %v", err)
	}

	name := *nameParam
	if name == "" {
		return nil, fmt.Errorf("missing required parameter: name")
	}

	path := *pathParam
	projectPath, err := file.ResolveProjectPath(path, name)
	if err != nil {
		return nil, fmt.Errorf("invalid project path: %v", err)
	}

	typStr := *typeParam
	typ, prs := SUPPORTED_PROJECT_TYPES[typStr]
	if !prs {
		return nil, fmt.Errorf("invalid project type: %s. Valid project types are: %s", typStr, SUPPORTED_PROJECT_TYPES_STR)
	}

	return &NewCommand{name, typ, projectPath}, nil
}

type NewCommand struct {
	name string
	typ  ProjectType
	path string
}

func (c *NewCommand) Invoke() error {
	return nil
}
