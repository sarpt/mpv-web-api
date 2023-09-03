package utils

import (
	"os"
	"path/filepath"
)

var (
	defaultAppDirName = ".mwa"
	subdirs           = []string{
		"playlists",
	}
)

func HandleAppDir(appDir string) (string, error) {
	if appDir == "" {
		appDir = getDefaultAppDir()
	}

	err := ensureAppDirs(appDir)
	return appDir, err
}

func ensureAppDirs(basePath string) error {
	for _, subdir := range subdirs {
		dirPath := filepath.Join(basePath, subdir)
		err := os.MkdirAll(dirPath, 0750)
		if err != nil {
			return err
		}
	}

	return nil
}

func getDefaultAppDir() string {
	var appPathDefaultBase string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		appPathDefaultBase = os.TempDir()
	} else {
		appPathDefaultBase = homeDir
	}

	return filepath.Join(appPathDefaultBase, defaultAppDirName)
}
