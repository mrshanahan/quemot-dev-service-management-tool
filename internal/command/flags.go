package command

import (
	"flag"
	"fmt"

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

func UseServerConfigFlags(fs *flag.FlagSet) ServerConfigFlags {
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
	remoteServiceDirectoryParam := fs.String(
		"remote-service-directory",
		"",
		"Path on the remote server to use as the base directory for smt services. Overrides property in config.",
	)
	return ServerConfigFlags{
		ConfigPath:             configPathParam,
		Server:                 serverParam,
		Hostname:               hostnameParam,
		SshUsername:            sshUsernameParam,
		SshKeyFilePath:         sshKeyFilePathParam,
		RemoteServiceDirectory: remoteServiceDirectoryParam,
	}
}

func ValidateServerConfigFlags(s ServerConfigFlags) error {
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

	serverCfg, prs := cfg.Servers[server]
	if !prs {
		return fmt.Errorf("no server config exists for specified server %s", server)
	}
	serverConfig = serverCfg

	hostname := *s.Hostname
	if hostname == "" {
		hostname = serverConfig.Hostname
		if hostname == "" {
			return fmt.Errorf("no hostname specified for server %s", server)
		}
	}

	sshUsername := *s.SshUsername
	if sshUsername == "" {
		sshUsername = serverConfig.SshUsername
		if sshUsername == "" {
			return fmt.Errorf("no SSH username specified for server %s", server)
		}
	}

	sshKeyFilePath := *s.SshKeyFilePath
	if sshKeyFilePath == "" {
		sshKeyFilePath = serverConfig.SshKeyFilePath
		if sshKeyFilePath == "" {
			return fmt.Errorf("no SSH key file path specified for server %s", server)
		}
	}

	remoteServiceDirectory := *s.RemoteServiceDirectory
	if remoteServiceDirectory == "" {
		// This should always be non-empty. It should have a default value of config.DefaultServiceDirectory.
		remoteServiceDirectory = serverConfig.RemoteServiceDirectory
		if remoteServiceDirectory == "" {
			return fmt.Errorf("remote service directory is empty for server %s - this should not happen!", server)
		}
	}

	*s.Server = server
	*s.Hostname = hostname
	*s.SshUsername = sshUsername
	*s.SshKeyFilePath = sshKeyFilePath
	*s.RemoteServiceDirectory = remoteServiceDirectory

	return nil
}
