package command

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"

	"github.com/mrshanahan/deploy-assets/pkg/transport"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/install"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/sshclient"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

// TODO: This should also accept the normal config arguments (install + register)

type InstallCommandSpec struct {
	Args []string
}

type InstallCommand struct {
	installDir     string
	hostname       string
	sshUsername    string
	sshKeyFilePath string
	force          bool
}

func (s *InstallCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})

	installDirParam := fs.String(
		"install-dir",
		"",
		fmt.Sprintf("Directory where executable should be placed. Defaults to %s", install.DefaultInstallDir),
	)
	forceParam := fs.Bool(
		"force",
		false,
		"If specified, create install directory if it does not exist (otherwise install will fail)",
	)

	serverConfigFlags := UseServerConfigFlags(fs)

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

	if err := ValidateServerConfigFlags(serverConfigFlags); err != nil {
		return nil, err
	}

	if *debugParam {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	return &InstallCommand{
		installDir:     *installDirParam,
		hostname:       *serverConfigFlags.Hostname,
		sshUsername:    *serverConfigFlags.SshUsername,
		sshKeyFilePath: *serverConfigFlags.SshKeyFilePath,
		force:          *forceParam,
	}, nil
}

func (c *InstallCommand) Invoke() error {
	sshExecutor, err := sshclient.CreateSshExecutor(c.hostname, c.sshUsername, c.sshKeyFilePath, "")
	if err != nil {
		return err
	}

	transport, err := transport.NewScpTransport("smt-scp", c.hostname, c.sshUsername, c.sshKeyFilePath, "")
	if err != nil {
		return err
	}

	if err := install.InstallSmt(sshExecutor, transport, c.installDir, c.force); err != nil {
		if errors.Is(err, install.ErrNoInstallDir) {
			return fmt.Errorf("install directory %s not located on remote server - ensure it exists or pass the -force flag to create it", c.installDir)
		}
		return fmt.Errorf("failed to install smt on remote server: %w", err)
	}

	return nil
}
