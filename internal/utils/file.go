package utils

import (
	"io"
	"os"
	"path/filepath"
)

// IsSymlink checks if a file is a symlink
func IsSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

// CopyFile copies a file from src to dst
// src and dst are file paths
func CopyFile(src string, dst string, followSymlinks bool) error {
	// Is it symlink
	isSymlink, err := IsSymlink(src)
	if err != nil {
		return err
	}
	if isSymlink && !followSymlinks {
		return nil
	}

	// Copy file
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Flush the destination file to ensure all data is written
	err = destinationFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

// CopyTree recursively copies a directory tree from src to dst
func CopyTree(src string, dst string, followSymlinks bool) error {
	stack := [][2]string{{src, dst}}
	for len(stack) > 0 {
		// Pop the stack
		current := stack[0][0]
		currentDst := stack[0][1]
		stack = stack[1:]

		// Is it symlink
		isSymlink, err := IsSymlink(current)
		if err != nil {
			return err
		}
		if isSymlink && !followSymlinks {
			continue
		}

		// Is it a file
		fileInfo, err := os.Stat(current)
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			err := CopyFile(current, currentDst, followSymlinks)
			// If file already exists, skip
			if os.IsExist(err) {
				continue
			}
			if err != nil {
				return err
			}
		} else { // Else it is a directory
			// Mkdir the dir
			err := os.MkdirAll(currentDst, os.ModePerm)
			if err != nil && !os.IsExist(err) {
				return err
			}
			// Add its children path to the stack
			children, err := os.ReadDir(current)
			if err != nil {
				return err
			}
			for _, child := range children {
				// Path is current join child
				childPath := filepath.Join(current, child.Name())
				childDst := filepath.Join(currentDst, child.Name())
				stack = append(stack, [2]string{childPath, childDst})
			}
		}
	}
	return nil
}

// IsDirEmpty checks if a directory is empty
func IsDirEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(files) == 0, nil
}

// Contains checks if a slice contains an item
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Remove removes an item from a slice
func Remove(slice []string, item string) []string {
	for i, s := range slice {
		if s == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
