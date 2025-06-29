package stdiorunner

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CreateTarGzArchive(sourceDir, targetFile string) error {
	// Create the output .tar.gz file
	outFile, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Get absolute path of the archive to exclude it during walk
	archiveAbsPath, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("failed to resolve archive path: %w", err)
	}

	err = filepath.Walk(sourceDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %w", path, err)
		}

		// Skip symlinks and non-regular files
		if !fi.Mode().IsRegular() && !fi.IsDir() {
			return nil
		}

		// Skip the archive file itself
		absPath, err := filepath.Abs(path)
		if err == nil && absPath == archiveAbsPath {
			return nil
		}

		// Compute archive-relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}
		if strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("invalid path escapes source dir: %s", relPath)
		}
		if relPath == "." {
			return nil // skip root directory entry itself
		}

		// Create and write header
		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write header for %s: %w", relPath, err)
		}

		// If directory, no content to copy
		if fi.IsDir() {
			return nil
		}

		// Copy file contents
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create tar.gz archive: %w", err)
	}

	return nil
}
