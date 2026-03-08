package install

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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
			return fmt.Errorf("[%s] failed to create install directory %s: %w", remote.Name(), installDir, err)
		}
	}

	// Always gets installed as smt
	dstPath := filepath.Join(installDir, "smt")
	stdoutRaw, _, err := remote.ExecuteShell(fmt.Sprintf("(test -e '%s' && echo 'installed') || echo 'not-installed'", dstPath))
	if err != nil {
		return fmt.Errorf("[%s] failed to check for existing executable: %w", remote.Name(), err)
	}

	stdout := strings.Trim(stdoutRaw, " \n")
	if stdout == "not-installed" || force {
		slog.Info("beginning installing of smt", "server", remote.Name(), "path", dstPath, "executable-state", stdout, "force", force)

		srcPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("[local] could not get path of current executable: %w", err)
		}

		local := executor.NewLocalExecutor("local")

		slog.Debug("installing smt", "src-path", srcPath, "dst-path", dstPath)
		if err := transport.TransferFile(local, srcPath, remote, dstPath); err != nil {
			return fmt.Errorf("[local -> %s] failed to transfer file to remote: %w", remote.Name(), err)
		}
	} else {
		slog.Info("skipping install of smt", "server", remote.Name(), "path", dstPath, "executable-state", stdout, "force", force)
	}

	_, _, err = remote.ExecuteCommand("which", "smt")
	if err != nil {
		slog.Warn("smt was installed but 'which smt' failed; smt is likely not available on ssh user's PATH",
			"server", remote.Name(),
			"dst-path", dstPath,
			"err", err)
	}

	_, _, err = remote.ExecuteCommand("mkdir", "-p", DefaultServicesDir)
	if err != nil {
		return fmt.Errorf("[%s] failed to create services directory %s: %w", remote.Name(), DefaultServicesDir, err)
	}

	if _, _, err := remote.ExecuteShell(fmt.Sprintf("test -f '%s' || (echo '{}' > '%s')", DefaultConfigFilePath, DefaultConfigFilePath)); err != nil {
		return fmt.Errorf("[%s] failed to check or create remote config file %s: %w", remote.Name(), DefaultConfigFilePath, err)
	}

	return nil
}
