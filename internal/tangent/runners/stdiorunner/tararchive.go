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

		if !shouldIncludeFile(fi, path, archiveAbsPath) {
			return nil
		}

		relPath, err := validateAndGetRelativePath(path, sourceDir)
		if err != nil {
			return err
		}

		if err := writeFileToArchive(tarWriter, fi, relPath, path); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create tar.gz archive: %w", err)
	}

	return nil
}

// shouldIncludeFile determines if a file should be included in the archive.
// It filters out symlinks, non-regular files, and the archive file itself.
func shouldIncludeFile(fi os.FileInfo, path, archiveAbsPath string) bool {
	// Skip symlinks and non-regular files
	if !fi.Mode().IsRegular() && !fi.IsDir() {
		return false
	}

	// Skip the archive file itself
	absPath, err := filepath.Abs(path)
	if err == nil && absPath == archiveAbsPath {
		return false
	}

	return true
}

// validateAndGetRelativePath computes the relative path from sourceDir to path,
// validates that it doesn't escape the source directory, and skips the root directory.
func validateAndGetRelativePath(path, sourceDir string) (string, error) {
	// Compute archive-relative path
	relPath, err := filepath.Rel(sourceDir, path)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("invalid path escapes source dir: %s", relPath)
	}
	if relPath == "." {
		return "", nil // skip root directory entry itself
	}
	return relPath, nil
}

// writeFileToArchive creates a tar header for the file and writes it to the archive.
// For directories, it only writes the header. For regular files, it also copies the content.
func writeFileToArchive(tarWriter *tar.Writer, fi os.FileInfo, relPath, filePath string) error {
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

	return copyFileContent(tarWriter, filePath, relPath)
}

// copyFileContent opens the file and copies its contents to the tar archive.
func copyFileContent(tarWriter *tar.Writer, filePath, _ string) error {
	// Copy file contents
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(tarWriter, f); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}
