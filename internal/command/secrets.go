package command

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/mrshanahan/go-utils/term"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/config"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/project"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/secrets"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/sshclient"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

type SecretsCommandSpec struct {
	Args []string
}

type SecretAction int

const (
	ListSecrets SecretAction = iota
	SetSecret
	RemoveSecret
	ShowSecret
)

var (
	validateSecretNamePatternString string         = "^[A-Z_0-9]+$"
	validateSecretNamePattern       *regexp.Regexp = regexp.MustCompile(validateSecretNamePatternString)
)

type SecretsCommand struct {
	projectConfig  *project.ProjectConfig
	name           string
	valueSet       bool
	value          string
	action         SecretAction
	hostname       string
	sshKeyFilePath string
	sshUsername    string
}

func (s *SecretsCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("secrets", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})

	// TODO: Common command sets (for connecting to remote server), hierarchical commands
	pathParam := fs.String(
		"path",
		"",
		"Path to the project to deploy. Defaults to current working directory.",
	)
	listParam := fs.Bool(
		"list",
		false,
		"(action) (default) List the existing secrets without values",
	)
	setParam := fs.Bool(
		"set",
		false,
		"(action) Set a secret entry to a new value",
	)
	removeParam := fs.Bool(
		"remove",
		false,
		"(action) Remove a secret entry",
	)
	showParam := fs.Bool(
		"show",
		false,
		"(action) Show a secret entry's value",
	)
	nameParam := fs.String(
		"name",
		"",
		"Name of the secret to show/modify (if relevant)",
	)
	valueParam := fs.String(
		"value",
		"",
		"Value of the secret to set (if relevant)",
	)
	configPathParam := fs.String(
		"config",
		"",
		"Path to deployment config file. Defaults to ~/.config/smt.config.",
	)
	serverParam := fs.String(
		"server",
		"",
		"Name of the server to deploy to, matching an entry in the config file. If not provided then directly-provided properties will be used.",
	)
	hostnameParam := fs.String(
		"hostname",
		"",
		"Hostname of the server to deploy to. Overrides property in config.",
	)
	sshUsernameParam := fs.String(
		"ssh-username",
		"",
		"Username to use for SSH connection. Overrides property in config.",
	)
	sshKeyFilePathParam := fs.String(
		"ssh-key-file",
		"",
		"Path to the SSH key file path. Overrides property in config.",
	)
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

	configPath := *configPathParam
	if configPath == "" {
		defaultPath, err := config.GetDefaultPath()
		if err != nil {
			return nil, fmt.Errorf("could not get default config file path: %w", err)
		}
		configPath = defaultPath
	}

	slog.Debug("loading config", "path", configPath)

	cfg, err := config.LoadConfig(configPath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to load config at %s: %w", configPath, err)
	}

	path := *pathParam
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get cwd: %w", err)
		}
		path = wd
	}
	projectConfigPath, err := project.GetProjectConfigPath(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("project path '%s' is not a %s file nor does it contain one", path, project.ProjectConfigName)
		}
		return nil, err
	}

	server := *serverParam

	var serverConfig *config.ServerConfig
	if server == "" {
		server = cfg.DefaultServer
		if server == "" {
			return nil, fmt.Errorf("no server specified and no default server in config")
		}
		serverCfg, prs := cfg.Servers[server]
		if !prs {
			return nil, fmt.Errorf("no server config exists for specified server %s", server)
		}
		serverConfig = serverCfg
	}

	hostname := *hostnameParam
	if hostname == "" {
		hostname = serverConfig.Hostname
		if hostname == "" {
			return nil, fmt.Errorf("no hostname specified for server %s", server)
		}
	}

	sshUsername := *sshUsernameParam
	if sshUsername == "" {
		sshUsername = serverConfig.SshUsername
		if sshUsername == "" {
			return nil, fmt.Errorf("no SSH username specified for server %s", server)
		}
	}

	sshKeyFilePath := *sshKeyFilePathParam
	if sshKeyFilePath == "" {
		sshKeyFilePath = serverConfig.SshKeyFilePath
		if sshKeyFilePath == "" {
			return nil, fmt.Errorf("no SSH key file path specified for server %s", server)
		}
	}

	actionParams := map[SecretAction]bool{
		ListSecrets:  *listParam,
		SetSecret:    *setParam,
		RemoveSecret: *removeParam,
		ShowSecret:   *showParam,
	}

	var actions []SecretAction
	for k, v := range actionParams {
		if v {
			actions = append(actions, k)
		}
	}

	if len(actions) > 1 {
		return nil, fmt.Errorf("multiple actions specified; please specify at most one")
	}

	var action SecretAction
	if len(actions) == 0 {
		action = ListSecrets
	} else {
		action = actions[0]
	}

	name := *nameParam
	if name == "" && (action == ShowSecret || action == SetSecret || action == RemoveSecret) {
		return nil, fmt.Errorf("secret name required for specified action")
	}
	if name != "" && !validateSecretNamePattern.Match([]byte(name)) {
		return nil, fmt.Errorf("invalid secret name %s (must match /%s/)", name, validateSecretNamePatternString)
	}

	valueSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "value" {
			valueSet = true
		}
	})

	projectConfig, err := project.LoadProjectConfig(projectConfigPath)
	if err != nil {
		return nil, err
	}

	if projectConfig.DockerSecretsVolume == "" {
		return nil, fmt.Errorf("no Docker secrets volume specified for this project; add a docker_secrets_volume entry to %s and try again", projectConfigPath)
	}

	return &SecretsCommand{
		projectConfig:  projectConfig,
		name:           name,
		valueSet:       valueSet,
		value:          *valueParam,
		action:         action,
		hostname:       hostname,
		sshKeyFilePath: sshKeyFilePath,
		sshUsername:    sshUsername,
	}, nil
}

func (c *SecretsCommand) Invoke() error {
	sshExecutor, err := sshclient.CreateSshExecutor(c.hostname, c.sshUsername, c.sshKeyFilePath, "")
	if err != nil {
		return err
	}

	secretVolumeName := c.projectConfig.DockerSecretsVolume
	secretsVolume, err := secrets.GetSecretsVolume(sshExecutor, secretVolumeName)
	if err != nil {
		return err
	}

	if secretsVolume == nil {
		prompt := fmt.Sprintf("Secrets volume %s does not exist on %s. Do you want to create it?", secretVolumeName, c.hostname)
		yes, err := utils.BinaryPrompt(prompt)
		if err != nil || !yes {
			return fmt.Errorf("user declined to create volume %s - cannot proceed with secrets management", secretVolumeName)
		}

		secretsVolume, err = secrets.EnsureSecretsVolume(sshExecutor, secretVolumeName, false)
		if err != nil {
			return err
		}
	}

	switch c.action {
	case ListSecrets:
		entries := secretsVolume.Secrets
		var remoteEntries, localEntries []string
		if c.name == "" {
			remoteEntries = entries
			localEntries = c.projectConfig.Secrets
		} else {
			remoteEntries = utils.Filter(entries, func(x string) bool { return c.name == x })
			localEntries = utils.Filter(c.projectConfig.Secrets, func(x string) bool { return c.name == x })
		}

		fmt.Println(utils.BuildComparisonTable("LOCAL", localEntries, "REMOTE", remoteEntries))
	case SetSecret:
		if !slices.Contains(c.projectConfig.Secrets, c.name) {
			c.projectConfig.Secrets = append(c.projectConfig.Secrets, c.name)
			if err := project.SaveProjectConfig(c.projectConfig); err != nil {
				return fmt.Errorf("failed to save project config with updated secrets: %w", err)
			}
		}

		var value string
		if !c.valueSet {
			valuesMatch := false
			var value1, value2 string
			for !valuesMatch {
				value1, err = term.PromptSensitive("Enter secret value")
				if err != nil {
					return err
				}
				value2, err = term.PromptSensitive("Enter value again")
				if err != nil {
					return err
				}
				valuesMatch = value1 == value2
				if !valuesMatch {
					fmt.Fprintf(os.Stderr, "values do not match; please enter again\n\n")
				}
			}
			value = value1
		} else {
			value = c.value
		}

		entries := secretsVolume.Secrets
		if !slices.Contains(entries, c.name) {
			slog.Info("secret not present in deployed service; adding", "secret", c.name, "server", c.hostname)
		} else {
			slog.Info("secret present in deployed service; updating", "secret", c.name, "server", c.hostname)
		}

		valueb64 := base64.StdEncoding.EncodeToString([]byte(value))
		_, stderr, err := sshExecutor.ExecuteShell(
			fmt.Sprintf("echo '%s' | docker run -i -v %s:/secrets:rw alpine sh -c \"base64 -d > /secrets/%s\"",
				valueb64,
				secretVolumeName,
				c.name))
		if err != nil {
			return fmt.Errorf("failed to update secret %s - check error output (stderr: %s): %w", c.name, stderr, err)
		}
	case RemoveSecret:
		if !slices.Contains(c.projectConfig.Secrets, c.name) {
			slog.Warn("secret not present in project config - checking deployed service", "secret", c.name)
		} else {
			c.projectConfig.Secrets = utils.Filter(c.projectConfig.Secrets, func(s string) bool { return s != c.name })
			if err := project.SaveProjectConfig(c.projectConfig); err != nil {
				return fmt.Errorf("failed to save project config with updated secrets: %w", err)
			}
		}

		entries := secretsVolume.Secrets
		if !slices.Contains(entries, c.name) {
			slog.Warn("secret not present in deployed service; skipping removal", "secret", c.name, "server", c.hostname)
			return nil
		}

		_, stderr, err := sshExecutor.ExecuteCommand("docker", "run", "-it", "--rm", "-v", fmt.Sprintf("%s:/secrets:rw", secretVolumeName), "alpine", "rm", fmt.Sprintf("/secrets/%s", c.name))
		if err != nil {
			return fmt.Errorf("failed to remove secret %s - check error output (stderr: %s): %w", c.name, stderr, err)
		}
	case ShowSecret:
		if !slices.Contains(c.projectConfig.Secrets, c.name) {
			return fmt.Errorf("secret %s not registered with project - ensure it is added", c.name)
		}
		entries := secretsVolume.Secrets
		if !slices.Contains(entries, c.name) {
			return fmt.Errorf("secret %s not present in deployed service at %s - deploy secret first", c.name, c.hostname)
		}
		stdout, stderr, err := sshExecutor.ExecuteCommand("docker", "run", "-i", "--rm", "-v", fmt.Sprintf("%s:/secrets", secretVolumeName), "alpine", "cat", fmt.Sprintf("/secrets/%s", c.name))
		if err != nil {
			return fmt.Errorf("failed to retrieve secret content - check error output (stderr: %s): %w", stderr, err)
		}
		fmt.Println(stdout)
	}
	return nil
}

func compareSecrets(local []string, remote []string) {
	if local == nil && remote == nil {
		fmt.Println(" LOCAL  REMOTE")
		return
	}

	sortedLocal := slices.Clone(local)
	slices.Sort(sortedLocal)
	sortedRemote := slices.Clone(remote)
	slices.Sort(sortedRemote)

	lenF := func(x string) int { return len(x) }
	maxSecretNameLen := int(math.Max(
		float64(slices.Max(utils.Map(local, lenF))),
		float64(slices.Max(utils.Map(remote, lenF)))))

	fmt.Printf("%sLOCAL  REMOTE\n", strings.Repeat(" ", maxSecretNameLen+1))
	fmt.Printf("%s-----  ------\n", strings.Repeat(" ", maxSecretNameLen+1))
	var i, j int
	for i < len(sortedLocal) && j < len(sortedRemote) {
		if sortedLocal[i] == sortedLocal[j] {
			secretName := sortedLocal[i]
			diffFromMax := maxSecretNameLen - len(secretName)
			fmt.Printf("%s%s  %s      %s\n", secretName, strings.Repeat(" ", diffFromMax+1), "X", "X")
			i += 1
			j += 1
		} else if sortedLocal[i] < sortedRemote[j] {
			secretName := sortedLocal[i]
			diffFromMax := maxSecretNameLen - len(secretName)
			fmt.Printf("%s%s  %s      %s\n", secretName, strings.Repeat(" ", diffFromMax+1), "X", " ")
			i += 1
		} else {
			secretName := sortedRemote[j]
			diffFromMax := maxSecretNameLen - len(secretName)
			fmt.Printf("%s%s  %s      %s\n", secretName, strings.Repeat(" ", diffFromMax+1), " ", "X")
			j += 1
		}
	}
	for i < len(sortedLocal) {
		secretName := sortedLocal[i]
		diffFromMax := maxSecretNameLen - len(secretName)
		fmt.Printf("%s%s  %s      %s\n", secretName, strings.Repeat(" ", diffFromMax+1), "X", " ")
		i += 1
	}
	for j < len(sortedRemote) {
		secretName := sortedRemote[j]
		diffFromMax := maxSecretNameLen - len(secretName)
		fmt.Printf("%s%s  %s      %s\n", secretName, strings.Repeat(" ", diffFromMax+1), " ", "X")
		j += 1
	}
}
