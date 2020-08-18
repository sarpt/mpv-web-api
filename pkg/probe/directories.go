package probe

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkippedFile holds information about the path of pile that is skipped from probing results and the reason for why it's skipped
type SkippedFile struct {
	Path string
	Err  error
}

// Directories takes list of directories that should be checked for movie files.
func Directories(paths []string) ([]Result, []SkippedFile) {
	var results []Result
	var skippedFiles []SkippedFile

	for _, directory := range paths {
		filepath.Walk(directory, func(path string, info os.FileInfo, err error) error { // TODO: add some kind of error handling
			if err != nil {
				skippedFiles = append(skippedFiles, SkippedFile{
					Path: path,
					Err:  err,
				})
				return nil
			}

			if info.IsDir() {
				return nil
			}

			result, err := File(path)
			if err != nil {
				skippedFiles = append(skippedFiles, SkippedFile{
					Path: path,
					Err:  fmt.Errorf("file probing unsuccessful: %w", err),
				})
				return nil
			}

			results = append(results, result)
			return nil
		})
	}

	return results, skippedFiles
}
