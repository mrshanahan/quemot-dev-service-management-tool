package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

const ProjectConfigName string = "smt.json"

type ProjectConfig struct {
	ProjectDir        string
	NginxConfFiles    []string
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	ImageNames        []string `json:"image_names"`
	ImageCompareLabel string   `json:"image_compare_label"`
	DockerComposePath string   `json:"docker_compose_path"`
	SystemctlFilesDir string   `json:"systemctl_files_dir"`
	NginxFilesDir     string   `json:"nginx_files_dir"`

	AdditionalAssets []AdditionalAsset `json:"additional_assets"`
}

type AdditionalAsset struct {
	Name    string `json:"name"`
	SrcPath string `json:"src_path"`

	// TODO: How do we make this work?
	DstPath   string `json:"dst_path"`
	Recursive bool   `json:"recursive"`
	Force     bool   `json:"force"`
}

func LoadProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project config file %s: %w", path, err)
	}
	config, err := parseProjectConfig(data)
	if err != nil {
		return nil, err
	}
	config.ProjectDir = filepath.Dir(path)

	nginxFilesDir := config.NginxFilesDir
	if nginxFilesDir != "" {
		nginxFilesDirFull := filepath.Join(config.ProjectDir, nginxFilesDir)
		prs, err := utils.DirExists(nginxFilesDirFull)
		if err != nil {
			return nil, fmt.Errorf("failed to read subdirectory '%s' of project: %w", nginxFilesDir, err)
		}
		if prs {
			files, err := os.ReadDir(nginxFilesDirFull)
			if err != nil {
				return nil, fmt.Errorf("failed to read nginx entries in %s: %w", nginxFilesDirFull, err)
			}
			nginxFilePaths := []string{}
			for _, f := range files {
				nginxFilePaths = append(nginxFilePaths, filepath.Join("nginx", f.Name()))
			}
			config.NginxConfFiles = nginxFilePaths
		} else {
			config.NginxConfFiles = nil
		}
	}

	return config, nil
}

func parseProjectConfig(data []byte) (*ProjectConfig, error) {
	var config *ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse project config: %w", err)
	}
	return config, nil
}

// Given either a project root directory or a project config path,
// returns the correct absolute project config path. Error is returned
// if the path is invalid, the directory does not contain an smt.json
// file, or the given path is not an smt.json file.
func GetProjectConfigPath(projectPath string) (string, error) {
	projectPathAbs, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	projectPathInfo, err := os.Stat(projectPathAbs)
	if err != nil {
		return "", fmt.Errorf("failed to get path info: %w", err)
	}

	if projectPathInfo.IsDir() {
		configPath := filepath.Join(projectPathAbs, ProjectConfigName)
		_, err := os.Stat(configPath)
		if err != nil {
			return "", fmt.Errorf("failed to get config file info: %w", err)
		}
		return configPath, nil
	}

	fileName := projectPathInfo.Name()
	if strings.ToLower(fileName) != ProjectConfigName {
		return "", fmt.Errorf("path is not a project config file (expected name %s, got %s)",
			ProjectConfigName,
			fileName)
	}

	return projectPathAbs, nil
}
