package database

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	
	"github.com/iotaledger/hive.go/ioutils"
)

// ReadTOMLFromFile reads TOML data from the file named by filename to data.
// ReadTOMLFromFile uses toml.Unmarshal to decode data. Data must be a pointer to a fixed-size value or a slice
// of fixed-size values.
func ReadTOMLFromFile(filename string, data interface{}) error {
	tomlData, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read TOML file %s: %w", filename, err)
	}
	return toml.Unmarshal(tomlData, data)
}

// WriteTOMLToFile writes the TOML representation of data to a file named by filename.
// If the file does not exist, WriteTOMLToFile creates it with permissions perm
// (before umask); otherwise WriteTOMLToFile truncates it before writing, without changing permissions.
// WriteTOMLToFile uses toml.Marshal to encode data. Data must be a pointer to a fixed-size value or a slice
// of fixed-size values. An additional header can be passed.
func WriteTOMLToFile(filename string, data interface{}, perm os.FileMode, header ...string) (err error) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	tomlData, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to marshal data to TOML: %w", err)
	}

	if len(header) > 0 {
		if _, err := f.Write([]byte(header[0] + "\n")); err != nil {
			return fmt.Errorf("unable to write header to %s: %w", filename, err)
		}
	}

	if _, err := f.Write(tomlData); err != nil {
		return fmt.Errorf("unable to write TOML data to %s: %w", filename, err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("unable to fsync file content to %s: %w", filename, err)
	}

	return nil
}

// DatabaseExists checks if the database folder exists and is not empty.
func DatabaseExists(dbPath string) (bool, error) {

	dirExists, err := ioutils.PathExists(dbPath)
	if err != nil {
		return false, fmt.Errorf("unable to check database path (%s): %w", dbPath, err)
	}
	if !dirExists {
		return false, nil
	}

	// directory exists, but maybe database doesn't exist.
	// check if the directory is empty (needed for example in docker environments)
	dirEmpty, err := DirectoryEmpty(dbPath)
	if err != nil {
		return false, fmt.Errorf("unable to check database path (%s): %w", dbPath, err)
	}

	return !dirEmpty, nil
}

// DirectoryEmpty returns whether the given directory is empty.
func DirectoryEmpty(dirPath string) (bool, error) {

	// check if the directory exists
	if _, err := os.Stat(dirPath); err != nil {
		return false, fmt.Errorf("unable to check directory (%s): %w", dirPath, err)
	}

	// check if the directory is empty
	if err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirPath == path {
			// skip the root folder itself
			return nil
		}

		return os.ErrExist
	}); err != nil {
		if !os.IsExist(err) {
			return false, fmt.Errorf("unable to check directory (%s): %w", dirPath, err)
		}

		// directory is not empty
		return false, nil
	}

	// directory is empty
	return true, nil
}
