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

// Directory takes a single directory path that should be checked for media files.
func Directory(path string) ([]Result, []SkippedFile) {
	var results []Result
	var skippedFiles []SkippedFile

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error { // TODO: add some kind of error handling
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

	return results, skippedFiles
}
