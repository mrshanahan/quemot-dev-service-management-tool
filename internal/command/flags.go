package command

import (
	"flag"
	"fmt"
	"slices"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/config"
)

type ServerConfigFlags struct {
	ConfigPath             *string
	Server                 *string
	Hostname               *string
	SshUsername            *string
	SshKeyFilePath         *string
	RemoteServiceDirectory *string
}

func UseServerConfigFlags(fs *flag.FlagSet, include ...string) *ServerConfigFlags {
	flags := &ServerConfigFlags{}
	flags.ConfigPath = fs.String(
		"config",
		"",
		"Path to deployment config file. Defaults to ~/.config/smt.config.",
	)
	flags.Server = fs.String(
		"server",
		"",
		"Name of the server to deploy to, matching an entry in the config file. If not provided then directly-provided properties will be used.",
	)
	if len(include) == 0 || slices.Contains(include, "hostname") {
		flags.Hostname = fs.String(
			"hostname",
			"",
			"Hostname of the server to deploy to. Overrides property in config.",
		)
	}
	if len(include) == 0 || slices.Contains(include, "ssh-username") {
		flags.SshUsername = fs.String(
			"ssh-username",
			"",
			"Username to use for SSH connection. Overrides property in config.",
		)
	}
	if len(include) == 0 || slices.Contains(include, "ssh-key-file") {
		flags.SshKeyFilePath = fs.String(
			"ssh-key-file",
			"",
			"Path to the SSH key file path. Overrides property in config.",
		)
	}
	if len(include) == 0 || slices.Contains(include, "remote-service-directory") {
		flags.RemoteServiceDirectory = fs.String(
			"remote-service-directory",
			"",
			"Path on the remote server to use as the base directory for smt services. Overrides property in config.",
		)
	}

	return flags
}

func ValidateServerConfigFlags(s *ServerConfigFlags) error {
	configPath := *s.ConfigPath
	if configPath == "" {
		defaultPath, err := config.GetDefaultPath()
		if err != nil {
			return fmt.Errorf("could not get default config file path: %w", err)
		}
		configPath = defaultPath
	}

	cfg, err := config.LoadConfig(configPath, true)
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %w", configPath, err)
	}

	server := *s.Server

	var serverConfig *config.ServerConfig
	if server == "" {
		server = cfg.DefaultServer
		if server == "" {
			return fmt.Errorf("no server specified and no default server in config")
		}
	}
	*s.Server = server

	serverCfg, prs := cfg.Servers[server]
	if !prs {
		return fmt.Errorf("no server config exists for specified server %s", server)
	}
	serverConfig = serverCfg

	if s.Hostname != nil {
		hostname := *s.Hostname
		if hostname == "" {
			hostname = serverConfig.Hostname
			if hostname == "" {
				return fmt.Errorf("no hostname specified for server %s", server)
			}
		}
		*s.Hostname = hostname
	}

	if s.SshUsername != nil {
		sshUsername := *s.SshUsername
		if sshUsername == "" {
			sshUsername = serverConfig.SshUsername
			if sshUsername == "" {
				return fmt.Errorf("no SSH username specified for server %s", server)
			}
		}
		*s.SshUsername = sshUsername
	}

	if s.SshKeyFilePath != nil {
		sshKeyFilePath := *s.SshKeyFilePath
		if sshKeyFilePath == "" {
			sshKeyFilePath = serverConfig.SshKeyFilePath
			if sshKeyFilePath == "" {
				return fmt.Errorf("no SSH key file path specified for server %s", server)
			}
		}
		*s.SshKeyFilePath = sshKeyFilePath
	}

	if s.RemoteServiceDirectory != nil {
		remoteServiceDirectory := *s.RemoteServiceDirectory
		if remoteServiceDirectory == "" {
			// This should always be non-empty. It should have a default value of config.DefaultServiceDirectory.
			remoteServiceDirectory = serverConfig.RemoteServiceDirectory
			if remoteServiceDirectory == "" {
				return fmt.Errorf("remote service directory is empty for server %s - this should not happen!", server)
			}
		}
		*s.RemoteServiceDirectory = remoteServiceDirectory
	}

	return nil
}
