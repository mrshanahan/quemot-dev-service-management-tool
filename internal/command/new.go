package command

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
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
                        Must match pattern: %s
    
        -type <string>  Type of project to create.
                        Available types: %s
`, validateNamePattern, SUPPORTED_PROJECT_TYPES_STR)
}

var (
	validateNamePattern string         = `^[a-zA-Z0-9\-]*[a-zA-Z0-9]$`
	ValidateName        *regexp.Regexp = regexp.MustCompile(validateNamePattern)
)

func (s *NewCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})
	nameParam := fs.String("name", "", "")
	pathParam := fs.String("path", "", "")
	typeParam := fs.String("type", "service", "")
	debugParam := fs.Bool("debug", false, "")

	if err := fs.Parse(s.Args); err != nil {
		if err == flag.ErrHelp {
			return nil, err
		}
		return nil, fmt.Errorf("failed to parse command line: %v", err)
	}

	if *debugParam {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	name := *nameParam
	if name == "" {
		return nil, fmt.Errorf("missing required parameter: name")
	}
	if !ValidateName.MatchString(name) {
		return nil, fmt.Errorf("invalid name: %s", name)
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
	name        string
	projectType ProjectType
	path        string
}

//go:embed templates
var templates embed.FS

func (c *NewCommand) Invoke() error {
	if _, _, err := utils.ExecuteCommand("which", "go"); err != nil {
		return fmt.Errorf("go CLI is not available on PATH; ensure it is available & try again")
	}

	switch c.projectType {
	case SERVICE:
		if err := os.MkdirAll(c.path, 0o700); err != nil {
			return err
		}

		envVarPrefix := strings.ReplaceAll(strings.ToUpper(c.name), "-", "_")
		dockerImageName := fmt.Sprintf("quemot-dev/%s", c.name)
		hostname := fmt.Sprintf("%s.quemot.dev", c.name)
		variables := map[string]string{
			"NAME":              c.name,
			"DOCKER_IMAGE_NAME": dockerImageName,
			"HOSTNAME":          hostname,
			"ENVVAR_PREFIX":     envVarPrefix,
			"API_PORT_DEFAULT":  "8080",
		}
		if err := file.CopyTemplate(templates, "templates/service", c.path, variables); err != nil {
			return fmt.Errorf("failed to copy project template to %s: %w", c.path, err)
		}

		// TODO: Clean up if shit hits the fan

		moduleName := fmt.Sprintf("github.com/mrshanahan/%s", c.name)
		if _, _, err := utils.ExecuteCommandInDir(c.path, "go", "mod", "init", moduleName); err != nil {
			return fmt.Errorf("failed to initialize go module: %w", err)
		}

		packagesBytes, err := templates.ReadFile("templates/service/IGNORE__packages.txt")
		if err != nil {
			return fmt.Errorf("failed to load go package requirements: %w", err)
		}

		packages := strings.Split(string(packagesBytes), "\n")
		for _, p := range packages {
			if _, _, err := utils.ExecuteCommandInDir(c.path, "go", "get", "-u", p); err != nil {
				return fmt.Errorf("failed to install package %s: %w", p, err)
			}
		}

		if _, _, err := utils.ExecuteCommandInDir(c.path, "go", "mod", "tidy"); err != nil {
			return fmt.Errorf("failed to tidy go module: %w", err)
		}
	}
	return nil
}
