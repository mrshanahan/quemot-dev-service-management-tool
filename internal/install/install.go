package install

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mrshanahan/deploy-assets/pkg/config"
	"github.com/mrshanahan/deploy-assets/pkg/executor"
)

const (
	DefaultInstallDir     string = "/usr/local/bin"
	DefaultServicesDir    string = "/etc/smt"
	DefaultConfigFilePath string = "/etc/smt/smt.config"
)

var (
	ErrNoInstallDir error = fmt.Errorf("install directory does not exist")
)

// TODO: Wrap these errors up?
func InstallSmt(remote config.Executor, transport config.Transport, installDir string, force bool) error {
	if installDir == "" {
		installDir = DefaultInstallDir
	}

	if !force {
		_, _, err := remote.ExecuteCommand("test", "-d", installDir)
		if err != nil {
			return ErrNoInstallDir
		}
	} else {
		_, _, err := remote.ExecuteShell(fmt.Sprintf("mkdir -p '%s'", installDir))
		if err != nil {
			return fmt.Errorf("failed to create install directory %s: %w", installDir, err)
		}
	}

	// Always gets installed as smt
	dstPath := filepath.Join(installDir, "smt")

	srcPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not get path of current executable: %w", err)
	}

	local := executor.NewLocalExecutor("local")

	slog.Debug("installing", "src-path", srcPath, "dst-path", dstPath)
	if err := transport.TransferFile(local, srcPath, remote, dstPath); err != nil {
		return fmt.Errorf("failed to transfer file to remote: %w", err)
	}

	_, _, err = remote.ExecuteCommand("which", "smt")
	if err != nil {
		slog.Warn("smt was installed but 'which smt' failed; smt is likely not available on ssh user's PATH",
			"dst-path", dstPath,
			"err", err)
	}

	_, _, err = remote.ExecuteCommand("mkdir", "-p", DefaultServicesDir)
	if err != nil {
		return fmt.Errorf("failed to create services directory %s: %w", DefaultServicesDir, err)
	}

	_, _, err = remote.ExecuteCommand("touch", DefaultConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to create smt config file %s: %w", DefaultConfigFilePath, err)
	}

	return nil
}
