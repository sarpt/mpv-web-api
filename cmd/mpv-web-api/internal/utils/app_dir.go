package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

var (
	appDirId          = "mwa"
	defaultAppDirName = fmt.Sprintf(".%s", appDirId)
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

func GetCachePath(dir string) (string, error) {
	if len(dir) > 0 {
		return dir, nil
	}

	cacheDirPath, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("could not open user cache dir: %w\n", err)
	}

	return path.Join(cacheDirPath, appDirId), nil
}
