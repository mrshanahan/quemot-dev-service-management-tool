package secrets

import (
	"fmt"
	"log/slog"
	"strings"

	deploy "github.com/mrshanahan/deploy-assets/pkg/config"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

type SecretsVolume struct {
	Name    string
	Secrets []string
}

// Gets the Docker secret volume with the given name from the server pointed to by the executor.
// Returns a non-nil pointer to the volume if it exists, a nil pointer if it does not, and an error
// if any of the intermediate Docker commands fail.
func GetSecretsVolume(sshExecutor deploy.Executor, secretVolume string) (*SecretsVolume, error) {
	stdout, stderr, err := sshExecutor.ExecuteCommand("docker", "volume", "ls", "--filter", fmt.Sprintf("name=%s", secretVolume), "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve info for Docker volume %s (stdout=%s, stderr=%s): %w", secretVolume, stdout, stderr, err)
	}
	if strings.TrimSpace(stdout) != "" {
		stdout, stderr, err := sshExecutor.ExecuteCommand("docker", "run", "-a", "stdout", "--rm", "-v", fmt.Sprintf("%s:/secrets", secretVolume), "alpine", "ls", "-1", "/secrets")
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve secrets - check error output (stderr: %s): %w", stderr, err)
		}
		entries := utils.Filter(strings.Split(stdout, "\n"), func(x string) bool { return x != "" })
		return &SecretsVolume{secretVolume, entries}, nil
	}
	return nil, nil
}

func EnsureSecretsVolume(sshExecutor deploy.Executor, secretVolume string, dryRun bool) (*SecretsVolume, error) {
	stdout, stderr, err := sshExecutor.ExecuteCommand("docker", "volume", "ls", "--filter", fmt.Sprintf("name=%s", secretVolume), "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve info for Docker volume %s (stdout=%s, stderr=%s): %w", secretVolume, stdout, stderr, err)
	}
	if strings.TrimSpace(stdout) == "" {
		if dryRun {
			slog.Info("DRY RUN: creating Docker secrets volume", "name", secretVolume)
		} else {
			stdout, stderr, err := sshExecutor.ExecuteCommand("docker", "volume", "create", secretVolume)
			if err != nil {
				return nil, fmt.Errorf("failed to create Docker volume %s (stdout=%s, stderr=%s): %w", secretVolume, stdout, stderr, err)
			}
		}
	} else {
		slog.Debug("Docker secrets volume already exists", "name", secretVolume)
	}

	return &SecretsVolume{secretVolume, []string{}}, nil
}
