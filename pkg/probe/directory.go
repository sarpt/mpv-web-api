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
// TODO: Add error handling and some form of returning what files are skipped.
func Directory(path string, results chan<- Result) {
	defer close(results)
	var skippedFiles []SkippedFile

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
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

		results <- result
		return nil
	})
}
