package command

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
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

	AVAILABLE_FEATURES []string = []string{
		"auth",
		"static-assets",
		"database",
	}

	AVAILABLE_FEATURES_STR string = strings.Join(AVAILABLE_FEATURES, ", ")

	AVAILABLE_VARIABLES []string = []string{
		"API_PORT_DEFAULT",
		"DOMAIN_SUFFIX",
	}

	AVAILABLE_VARIABLES_STR string = strings.Join(AVAILABLE_VARIABLES, ", ")
)

type NewCommandSpec struct {
	Args []string
}

var (
	validateNamePattern string         = `^[a-zA-Z0-9\-]*[a-zA-Z0-9]$`
	ValidateName        *regexp.Regexp = regexp.MustCompile(validateNamePattern)
)

func (s *NewCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})
	nameParam := fs.String(
		"name",
		"",
		"Name of the project\n"+
			"This will be used as the repository name where possible (see -path) and executable name where appropriate.\n"+
			fmt.Sprintf("Must match pattern: %s", validateNamePattern))
	pathParam := fs.String(
		"path",
		"",
		"Path to the new project\n"+
			"- If the path does not exist, the project will be created there (irrespective of -name).\n"+
			"- If the path is a directory, the project will be created as a subdirectory therein with the name from -name.\n"+
			"- All other scenarios result in an error.")
	typeParam := fs.String(
		"type",
		"service",
		"Type of project to create\n"+
			fmt.Sprintf("Available types: %s", SUPPORTED_PROJECT_TYPES_STR))
	varsParam := fs.String(
		"vars",
		"",
		"Comma-separated list of additional variables in the form of <var1>=<value1>,<var2>=<value2>,etc.\n"+
			fmt.Sprintf("Available variables: %s", AVAILABLE_VARIABLES_STR))
	// featuresParams := fs.String(
	// 	"features",
	// 	"auth",
	// 	"Additional features to include, varying by service.\n"+
	// 		fmt.Sprintf("All available features are: %s", AVAILABLE_FEATURES_STR))
	debugParam := fs.Bool("debug", false, "Set log level to debug")

	if err := fs.Parse(s.Args); err != nil {
		if err != flag.ErrHelp {
			utils.PrintErrf("error: %v\n", err)
		}
		// return nil, err
		fs.SetOutput(nil)
		fs.Usage()
		return nil, err
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

	vars := map[string]string{}
	varsStr := *varsParam
	if varsStr != "" {
		comps := strings.Split(varsStr, ",")
		for _, c := range comps {
			splitIdx := strings.Index(c, "=")
			if splitIdx <= 0 {
				return nil, fmt.Errorf("invalid variable declaration: %s", c)
			}
			variable, value := c[:splitIdx], c[splitIdx+1:]
			if !slices.Contains(AVAILABLE_VARIABLES, variable) {
				return nil, fmt.Errorf("unknown variable: %s", variable)
			}
			vars[variable] = value
		}
	}

	return &NewCommand{name, typ, projectPath, vars}, nil
}

type NewCommand struct {
	name         string
	projectType  ProjectType
	path         string
	varOverrides map[string]string
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

		domain := "quemot.dev"
		if domainOverride, prs := c.varOverrides["DOMAIN_SUFFIX"]; prs {
			domain = domainOverride
		}

		envVarPrefix := strings.ReplaceAll(strings.ToUpper(c.name), "-", "_")
		dockerImageName := fmt.Sprintf("quemot-dev/%s", c.name)
		hostname := fmt.Sprintf("%s.%s", c.name, domain)
		variables := map[string]file.VariableValue{
			"NEW_GUID()":        file.VarFunc(func() string { return uuid.NewString() }),
			"NAME":              file.VarValue(c.name),
			"NAME_UPPER":        file.VarValue(strings.ToUpper(c.name)),
			"DOCKER_IMAGE_NAME": file.VarValue(dockerImageName),
			"HOSTNAME":          file.VarValue(hostname),
			"ENVVAR_PREFIX":     file.VarValue(envVarPrefix),
			"API_PORT_DEFAULT":  file.VarValue("8080"),
		}
		for k, v := range c.varOverrides {
			variables[k] = file.VarValue(v)
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
