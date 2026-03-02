package command

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/config"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/sshclient"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

type ConfigCommandSpec struct {
	Args []string
}

type ConfigCommand struct {
	configPath             string
	server                 string
	hostname               string
	sshKeyFilePath         string
	sshUsername            string
	remoteServiceDirectory string
	setDefault             bool
	force                  bool
	action                 ConfigAction
}

type ConfigAction int

const (
	SetConfig ConfigAction = iota
	DeleteConfig
	ShowConfig
	ValidateConfig
)

const (
	DefaultServerName string = "default"
)

func (s *ConfigCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})

	// TODO: Turn some of these into globals, and abstract out reading from config

	serverConfigFlags := UseServerConfigFlags(fs)

	setDefaultParam := fs.Bool(
		"set-default",
		false,
		"If provided, set the given server as the default as well as perform any other operations.",
	)
	forceParam := fs.Bool(
		"force",
		false,
		"If provided, create config file if it does not exist; if not provided, exit with error instead",
	)
	setParam := fs.Bool(
		"set",
		false,
		"(action) Set a given server's properties",
	)
	deleteParam := fs.Bool(
		"delete",
		false,
		"(action) Delete the given entry (if it exists)",
	)
	showParam := fs.Bool(
		"show",
		false,
		"(action) Show existing config & exit",
	)
	validateParam := fs.Bool(
		"validate",
		false,
		"(action) Validate a server's config by dialing it",
	)

	if err := fs.Parse(s.Args); err != nil {
		if err != flag.ErrHelp {
			utils.PrintErrf("error: %v\n", err)
		}
		// return nil, err
		fs.SetOutput(nil)
		fs.Usage()
		return nil, err
	}

	// NB: Not calling ValidateServerConfigFlags here b/c we're not really checking any of them

	configPath := *serverConfigFlags.ConfigPath
	if configPath == "" {
		defaultPath, err := config.GetDefaultPath()
		if err != nil {
			return nil, fmt.Errorf("could not get default config file path: %w", err)
		}
		configPath = defaultPath
	}

	server := *serverConfigFlags.Server
	setDefault := *setDefaultParam

	actionParams := map[ConfigAction]bool{
		SetConfig:      *setParam,
		DeleteConfig:   *deleteParam,
		ShowConfig:     *showParam,
		ValidateConfig: *validateParam,
	}

	var actions []ConfigAction
	for k, v := range actionParams {
		if v {
			actions = append(actions, k)
		}
	}

	if len(actions) > 1 {
		return nil, fmt.Errorf("multiple actions specified; please specify at most one")
	}

	var action ConfigAction
	if len(actions) == 0 {
		action = ShowConfig
	} else {
		action = actions[0]
	}

	if action == DeleteConfig && server == "" {
		return nil, fmt.Errorf("-server is required when deleting an entry")
	}
	if action == DeleteConfig && setDefault {
		return nil, fmt.Errorf("cannot both -delete and -set-default an entry")
	}

	c := &ConfigCommand{
		configPath:             configPath,
		server:                 server,
		hostname:               *serverConfigFlags.Hostname,
		sshUsername:            *serverConfigFlags.SshUsername,
		sshKeyFilePath:         *serverConfigFlags.SshKeyFilePath,
		remoteServiceDirectory: *serverConfigFlags.RemoteServiceDirectory,
		setDefault:             setDefault,
		force:                  *forceParam,
		action:                 action,
	}
	return c, nil
}

func (c *ConfigCommand) Invoke() error {
	cfg, err := config.LoadConfig(c.configPath, c.force)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("config file does not exist at %s - use -force to create it", c.configPath)
		}
		return fmt.Errorf("failed to load config at %s: %w", c.configPath, err)
	}

	if c.action == ShowConfig {
		server := c.server
		if server == "" {
			server = DefaultServerName
			slog.Info("no server name provided, using default", "server", server)
		}
		entry, prs := cfg.Servers[server]
		if !prs {
			return fmt.Errorf("server %s not found", server)
		}
		fmt.Printf("%s:\n", server)
		fmt.Printf("    hostname:                 %s\n", entry.Hostname)
		fmt.Printf("    ssh_username:             %s\n", entry.SshUsername)
		fmt.Printf("    ssh_key_file_path:        %s\n", entry.SshKeyFilePath)
		fmt.Printf("    remote_service_directory: %s\n", entry.RemoteServiceDirectory)
		fmt.Println()

		return nil
	}

	if c.action == DeleteConfig {
		if cfg.DefaultServer == c.server {
			slog.Warn("deleting default server - default server is now unset", "server", c.server)
			cfg.DefaultServer = ""
		}

		delete(cfg.Servers, c.server)
		return config.SaveConfig(c.configPath, cfg)
	}

	if c.action == ValidateConfig {
		server := c.server
		if server == "" {
			server = DefaultServerName
			slog.Info("no server name provided, using default", "server", server)
		}
		entry, prs := cfg.Servers[server]
		if !prs {
			return fmt.Errorf("server %s not found", server)
		}

		client, err := sshclient.CreateSshClient(entry.Hostname, entry.SshUsername, entry.SshKeyFilePath, entry.SshKeyFilePassphrase)
		if err != nil {
			return err
		}
		client.Close()
		return nil
	}

	if c.action == SetConfig {
		server := c.server
		if server == "" {
			server = DefaultServerName
			slog.Info("no server name provided, using default", "server", server)
		}
		entry, prs := cfg.Servers[server]

		var hostname, sshUsername, sshKeyFilePath, remoteServiceDirectory string
		if prs {
			hostname = entry.Hostname
			sshUsername = entry.SshUsername
			sshKeyFilePath = entry.SshKeyFilePath
			remoteServiceDirectory = entry.RemoteServiceDirectory
		} else {
			entry = &config.ServerConfig{}
			cfg.Servers[server] = entry
		}

		if c.hostname == "" {
			hostname, err = getInput("Enter hostname", entry.Hostname)
			if err != nil {
				// TODO: Wrap it up?
				return err
			}
		} else {
			hostname = c.hostname
		}

		if c.sshUsername == "" {
			sshUsername, err = getInput("Enter SSH username", entry.SshUsername)
			if err != nil {
				// TODO: Wrap it up?
				return err
			}
		} else {
			sshUsername = c.sshUsername
		}

		if c.sshKeyFilePath == "" {
			sshKeyFilePath, err = getInput("Enter SSH key file path", entry.SshKeyFilePath)
			if err != nil {
				// TODO: Wrap it up?
				return err
			}
		} else {
			sshKeyFilePath = c.sshKeyFilePath
		}

		if c.remoteServiceDirectory == "" {
			remoteServiceDirectory, err = getInput("Enter base service directory on remote", entry.RemoteServiceDirectory)
			if err != nil {
				// TODO: Wrap it up?
				return err
			}
		} else {
			remoteServiceDirectory = c.remoteServiceDirectory
		}

		entry.Hostname = hostname
		entry.SshUsername = sshUsername
		entry.SshKeyFilePath = sshKeyFilePath
		entry.RemoteServiceDirectory = remoteServiceDirectory

		if c.setDefault || cfg.DefaultServer == "" {
			cfg.DefaultServer = server
		}
	}

	return config.SaveConfig(c.configPath, cfg)
}

func getInput(prompt string, currentValue string) (string, error) {
	var fullPrompt string
	required := currentValue == ""
	if required {
		fullPrompt = fmt.Sprintf("%s [required]: ", prompt)
	} else {
		fullPrompt = fmt.Sprintf("%s [enter for current: %s]: ", prompt, currentValue)
	}

	input, err := promptForInput(fullPrompt, required)
	if err != nil {
		// TODO: Wrap it up?
		return "", err
	}
	if input == "" {
		return currentValue, nil
	}
	return input, nil
}

func promptForInput(prompt string, required bool) (string, error) {
	lineScanner := bufio.NewScanner(os.Stdin)
	input := ""
	for {
		fmt.Fprint(os.Stderr, prompt)
		if lineScanner.Scan() {
			input = lineScanner.Text()
			if input != "" || !required {
				return input, nil
			}
		} else {
			return "", fmt.Errorf("no input given")
		}
	}
}
