package command

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	deploy "github.com/mrshanahan/deploy-assets/pkg/config"
	"github.com/mrshanahan/deploy-assets/pkg/executor"
	"github.com/mrshanahan/deploy-assets/pkg/manifest"
	"github.com/mrshanahan/deploy-assets/pkg/provider"
	"github.com/mrshanahan/deploy-assets/pkg/runner"
	"github.com/mrshanahan/deploy-assets/pkg/transport"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/config"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/install"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/project"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/secrets"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/service"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/sshclient"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

type DeployCommandSpec struct {
	Args []string
}

type DeployCommand struct {
	projectConfig  *project.ProjectConfig
	hostname       string
	sshKeyFilePath string
	sshUsername    string
	s3BaseUrl      string
	dryRun         bool
	show           bool
	force          bool
}

const (
	REMOTE_SERVER_NAME string = "remote"
	LOCAL_SERVER_NAME  string = "local"
)

func (s *DeployCommandSpec) Build() (Command, error) {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	fs.SetOutput(&EmptyWriter{})

	// TODO: Turn some of these into globals, and abstract out reading from config

	pathParam := fs.String(
		"path",
		"",
		"Path to the project to deploy. Defaults to current working directory.",
	)
	s3BaseUrlParam := fs.String(
		"s3-base-url",
		"s3://quemot-dev-bucket/smt",
		"Base S3 URL to use for transfers to remote servers.",
	)

	serverConfigFlags := UseServerConfigFlags(fs)

	showParam := fs.Bool("show", false, "Do not actually copy anything, just show compiled manifest and exit")
	dryRunParam := fs.Bool("dry-run", false, "Do not actually copy anything, just calculate differences and exit")
	debugParam := fs.Bool("debug", false, "Set log level to debug")
	forceParam := fs.Bool("force", false, "Force-create directory structures when necessary. Amounts to setting '\"force\": true' on all file resources.")

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

	if err := ValidateServerConfigFlags(serverConfigFlags); err != nil {
		return nil, err
	}

	s3BaseUrl := *s3BaseUrlParam
	if !strings.HasPrefix(s3BaseUrl, "s3://") {
		return nil, fmt.Errorf("invalid S3 base URL: '%s' (must start with 's3://')", s3BaseUrl)
	}

	projectConfig, err := project.LoadProjectConfig(projectConfigPath)
	if err != nil {
		return nil, err
	}

	return &DeployCommand{
		projectConfig:  projectConfig,
		hostname:       *serverConfigFlags.Hostname,
		sshUsername:    *serverConfigFlags.SshUsername,
		sshKeyFilePath: *serverConfigFlags.SshKeyFilePath,
		s3BaseUrl:      s3BaseUrl,
		dryRun:         *dryRunParam,
		show:           *showParam,
		force:          *forceParam,
	}, nil
}

func (c *DeployCommand) Invoke() error {
	sshExecutor, err := sshclient.CreateSshExecutor(c.hostname, c.sshUsername, c.sshKeyFilePath, "")
	if err != nil {
		return err
	}

	if c.projectConfig.DockerSecretsVolume != "" {
		if _, err := secrets.EnsureSecretsVolume(sshExecutor, c.projectConfig.DockerSecretsVolume, c.dryRun); err != nil {
			return err
		}
	}

	serverConfig, err := config.LoadServerConfig(sshExecutor, install.DefaultConfigFilePath, true)
	if err != nil {
		return err
	}

	serviceDefn, err := serverConfig.LoadServiceDefinition(sshExecutor, c.projectConfig.Name, true)
	if err != nil {
		return err
	}
	if serviceDefn == nil {
		serviceDefn = service.NewServiceDefinition(c.projectConfig.Name)
		slog.Debug("service entry does not exist in server config; using default path", "name", c.projectConfig.Name, "new-path", serviceDefn.Path)
	}

	serviceDefn.ServiceConfig.Commands = c.projectConfig.Commands
	assets, err := buildAssets(serviceDefn, c.projectConfig, c.force)
	if err != nil {
		return fmt.Errorf("failed to build manifest assets list: %w", err)
	}
	manifest, err := buildManifest(c, assets)
	if err != nil {
		return fmt.Errorf("failed to build manifest: %w", err)
	}

	if c.show {
		fmt.Println("transport:")
		fmt.Println(manifest.Transport.Yaml(4))
		fmt.Println("servers:")
		for _, e := range manifest.Executors {
			fmt.Println(e.Yaml(4))
		}
		fmt.Println("assets:")
		for _, p := range manifest.Providers {
			fmt.Println(p.Yaml(4))
		}
		return nil
	}

	if err := runner.Execute(manifest, c.dryRun, false); err != nil {
		return err
	}

	if !c.dryRun {
		slog.Debug("updating server config with new service path", "path", serviceDefn.Path)
		serverConfig.Services[c.projectConfig.Name] = serviceDefn.Path
		if err := config.SaveServerConfig(sshExecutor, install.DefaultConfigFilePath, serverConfig); err != nil {
			return err
		}
	}

	return nil
}

func buildManifest(c *DeployCommand, assets []*deploy.ProviderConfig) (*manifest.Manifest, error) {
	sshKeyFilePassphrase := ""
	runElevated := true
	hostname := c.hostname
	if !strings.Contains(hostname, ":") {
		hostname = fmt.Sprintf("%s:22", hostname)
	}
	sshExecutor, err := executor.NewSSHExecutor(REMOTE_SERVER_NAME, hostname, c.sshUsername, c.sshKeyFilePath, sshKeyFilePassphrase, runElevated)
	if err != nil {
		return nil, fmt.Errorf("failed to build SSH executor: %w", err)
	}
	m := &manifest.Manifest{
		Transport: transport.NewS3Transport("smt", "s3://quemot-dev-bucket/smt"),
		Executors: map[string]deploy.Executor{
			LOCAL_SERVER_NAME:  executor.NewLocalExecutor("local"),
			REMOTE_SERVER_NAME: sshExecutor,
		},
		Providers: assets,
	}
	return m, nil
}

func buildAssets(serviceDefn *service.ServiceDefinition, c *project.ProjectConfig, force bool) ([]*deploy.ProviderConfig, error) {
	remoteDir := serviceDefn.Path
	assets := []*deploy.ProviderConfig{}
	dockerComposeAsset := &deploy.ProviderConfig{
		Provider:     provider.NewFileProvider("docker-compose-file", c.ProjectDir, c.DockerComposePath, filepath.Join(remoteDir, "docker-compose.yml"), false, force),
		Src:          LOCAL_SERVER_NAME,
		Dst:          REMOTE_SERVER_NAME,
		PostCommands: []*deploy.PostCommand{},
	}
	assets = append(assets, dockerComposeAsset)

	remoteSystemctlDir := filepath.Join(remoteDir, "systemctl")
	// enableSystemctlServicesCommand := fmt.Sprintf("systemctl enable %s --now && systemctl enable %s --now",
	// 	filepath.Join(remoteSystemctlDir, "*.service"),
	// 	filepath.Join(remoteSystemctlDir, "*.timer"))
	// TODO: Make this smarter, and abstract out service registration
	enableSystemctlServicesCommand := fmt.Sprintf("systemctl enable %s --now",
		filepath.Join(remoteSystemctlDir, "*.service"))
	systemctlInstallFilesAsset := &deploy.ProviderConfig{
		Provider: provider.NewFileProvider("systemctl-install-files", c.ProjectDir, c.SystemctlFilesDir, remoteSystemctlDir, true, force),
		Src:      LOCAL_SERVER_NAME,
		Dst:      REMOTE_SERVER_NAME,
		PostCommands: []*deploy.PostCommand{
			{
				Command: enableSystemctlServicesCommand,
				Trigger: "on_changed",
			},
		},
	}
	assets = append(assets, systemctlInstallFilesAsset)

	systemctlServiceName := fmt.Sprintf("%s.service", c.Name)
	dockerImagesAsset := &deploy.ProviderConfig{
		Provider: provider.NewDockerProvider("docker-images", c.ImageNames, c.ImageCompareLabel),
		Src:      LOCAL_SERVER_NAME,
		Dst:      REMOTE_SERVER_NAME,
		PostCommands: []*deploy.PostCommand{
			{
				Command: fmt.Sprintf("systemctl restart %s", systemctlServiceName),
				Trigger: "on_changed",
			},
		},
	}
	assets = append(assets, dockerImagesAsset)

	if c.NginxConfFiles != nil {
		remoteNginxSitesAvailable := "/etc/nginx/sites-available"
		remoteNginxSitesEnabled := "/etc/nginx/sites-enabled"

		// Do a separate asset for each conf file to properly handle the automatic linking
		// from sites-available to sites-enabled without knowing anything about the filenames
		for i, f := range c.NginxConfFiles {
			remoteAvailablePath := filepath.Join(remoteNginxSitesAvailable, filepath.Base(f))
			remoteEnabledPath := filepath.Join(remoteNginxSitesEnabled, filepath.Base(f))
			confFileAsset := &deploy.ProviderConfig{
				Provider: provider.NewFileProvider(fmt.Sprintf("nginx-conf-files-%02d", i+1), c.ProjectDir, f, remoteAvailablePath, false, force),
				Src:      LOCAL_SERVER_NAME,
				Dst:      REMOTE_SERVER_NAME,
				PostCommands: []*deploy.PostCommand{
					{
						// TODO: Real escaping, probably
						Command: fmt.Sprintf("ln -s '%s' '%s'",
							remoteAvailablePath,
							remoteEnabledPath),
						Trigger: "on_created",
					},
					{
						Command: "nginx -t && nginx -s reload",
						Trigger: "on_changed",
					},
				},
			}
			assets = append(assets, confFileAsset)
		}
	}

	serviceConfigJson, err := json.MarshalIndent(serviceDefn.ServiceConfig, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize service config to JSON: %w", err)
	}
	serviceConfigPath := service.GetDefaultConfigPath(serviceDefn.Path)
	serviceConfigAsset := &deploy.ProviderConfig{
		Provider:     provider.NewLiteralProvider("service-config-json", string(serviceConfigJson), serviceConfigPath),
		Src:          LOCAL_SERVER_NAME,
		Dst:          REMOTE_SERVER_NAME,
		PostCommands: []*deploy.PostCommand{},
	}
	assets = append(assets, serviceConfigAsset)

	for _, a := range c.AdditionalAssets {
		additionalAsset := &deploy.ProviderConfig{
			Provider:     provider.NewFileProvider(a.Name, c.ProjectDir, a.SrcPath, a.DstPath, a.Recursive, a.Force),
			Src:          LOCAL_SERVER_NAME,
			Dst:          REMOTE_SERVER_NAME,
			PostCommands: []*deploy.PostCommand{},
		}
		assets = append(assets, additionalAsset)
	}

	return assets, nil
}
