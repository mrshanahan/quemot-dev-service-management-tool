package command

import (
	"flag"
	"fmt"
	"log/slog"

	"github.com/mrshanahan/deploy-assets/pkg/config"
	"github.com/mrshanahan/deploy-assets/pkg/executor"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/install"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/sshclient"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"

	serverconfig "github.com/mrshanahan/quemot-dev-service-management-tool/internal/config"
)

type ServiceCommandSpec struct {
	Args []string
}

type ServiceCommand struct {
	local                  bool
	name                   string
	action                 ServiceAction
	hostname               string
	sshKeyFilePath         string
	sshUsername            string
	remoteServiceDirectory string
}

type ServiceAction int

const (
	ListServices ServiceAction = iota
	StartService
	StopService
	RestartService
	RemoveService
)

func (s *ServiceCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("service", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})

	// TODO: Common command sets (for connecting to remote server), hierarchical commands
	localParam := fs.Bool(
		"local",
		false,
		"Run commands locally instead of remotely, i.e. if on a remote machine. Wins over -server.",
	)
	listParam := fs.Bool(
		"list",
		false,
		"(action) (default) List the services running in the target environment, filter to a single one with -name",
	)
	startParam := fs.Bool(
		"start",
		false,
		"(action) Starts a service provided by -name, noop if already started",
	)
	stopParam := fs.Bool(
		"stop",
		false,
		"(action) Stops a service provided by -name, noop if already stopped",
	)
	restartParam := fs.Bool(
		"restart",
		false,
		"(action) Restarts a service provided by -name",
	)
	removeParam := fs.Bool(
		"remove",
		false,
		"(action) Removes a service provided by -name, noop if it doesn't exist",
	)
	nameParam := fs.String(
		"name",
		"",
		"Name of the service",
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

	if *debugParam {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	cmd := &ServiceCommand{
		local:                  *localParam,
		hostname:               "",
		sshUsername:            "",
		sshKeyFilePath:         "",
		remoteServiceDirectory: "",
	}

	if !*localParam {
		if err := ValidateServerConfigFlags(serverConfigFlags); err != nil {
			return nil, err
		}
		cmd.hostname = *serverConfigFlags.Hostname
		cmd.sshUsername = *serverConfigFlags.SshUsername
		cmd.sshKeyFilePath = *serverConfigFlags.SshKeyFilePath
	}

	actionParams := map[ServiceAction]bool{
		ListServices:   *listParam,
		StartService:   *startParam,
		StopService:    *stopParam,
		RestartService: *restartParam,
		RemoveService:  *removeParam,
	}

	var actions []ServiceAction
	for k, v := range actionParams {
		if v {
			actions = append(actions, k)
		}
	}

	if len(actions) > 1 {
		return nil, fmt.Errorf("multiple actions specified; please specify at most one")
	}

	var action ServiceAction
	if len(actions) == 0 {
		action = ListServices
	} else {
		action = actions[0]
	}

	name := *nameParam
	if name == "" && action != ListServices {
		return nil, fmt.Errorf("service name required for specified action")
	}

	cmd.action = action
	cmd.name = name

	return cmd, nil
}

func (c *ServiceCommand) Invoke() error {
	var exec config.Executor
	if c.local {
		exec = executor.NewLocalExecutor("local")
	} else {
		sshExec, err := sshclient.CreateSshExecutor(c.hostname, c.sshUsername, c.sshKeyFilePath, "")
		if err != nil {
			return err
		}
		exec = sshExec
	}

	serverConfig, err := serverconfig.LoadRemoteServerConfig(exec, install.DefaultConfigFilePath, true)
	if err != nil {
		return err
	}

	switch c.action {
	case ListServices:
		values := []map[string]string{}
		for k, v := range serverConfig.Services {
			values = append(values, map[string]string{
				"NAME": k,
				"PATH": v,
			})
		}
		fmt.Println(utils.BuildTable([]string{"NAME", "PATH"}, values))
	default:
		fmt.Println("not supported yet! Sorry!")
	}

	// TODO: How do we determine where these things live locally?

	return nil
}
