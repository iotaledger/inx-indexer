package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/iotaledger/hive.go/ioutils"
	"github.com/iotaledger/hive.go/logger"
)

type Engine string

const (
	EngineUnknown Engine = "unknown"
	EngineAuto    Engine = "auto"
	EngineSQLite  Engine = "sqlite"
)

type databaseInfo struct {
	Engine string `toml:"databaseEngine"`
}

// DatabaseEngineFromString parses an engine from a string.
// Returns an error if the engine is unknown.
func DatabaseEngineFromString(engineStr string) (Engine, error) {

	dbEngine := Engine(strings.ToLower(engineStr))

	switch dbEngine {
	case "":
		// no engine specified
		fallthrough
	case EngineAuto:
		return EngineAuto, nil
	case EngineSQLite:
		return EngineSQLite, nil
	default:
		return EngineUnknown, fmt.Errorf("unknown database engine: %s, supported engines: sqlite", dbEngine)
	}
}

// DatabaseEngineAllowed checks if the database engine is allowed.
func DatabaseEngineAllowed(dbEngine Engine, allowedEngines ...Engine) (Engine, error) {

	if len(allowedEngines) > 0 {
		supportedEngines := ""
		for i, allowedEngine := range allowedEngines {
			if i != 0 {
				supportedEngines += "/"
			}
			supportedEngines += string(allowedEngine)

			if dbEngine == allowedEngine {
				return dbEngine, nil
			}
		}

		return "", fmt.Errorf("unknown database engine: %s, supported engines: %s", dbEngine, supportedEngines)
	}

	switch dbEngine {
	case EngineSQLite:
	default:
		return "", fmt.Errorf("unknown database engine: %s, supported engines: sqlite", dbEngine)
	}

	return dbEngine, nil
}

// DatabaseEngineFromStringAllowed parses an engine from a string and checks if the database engine is allowed.
func DatabaseEngineFromStringAllowed(dbEngineStr string, allowedEngines ...Engine) (Engine, error) {

	dbEngine, err := DatabaseEngineFromString(dbEngineStr)
	if err != nil {
		return EngineUnknown, err
	}

	return DatabaseEngineAllowed(dbEngine, allowedEngines...)
}

// CheckDatabaseEngine checks if the correct database engine is used.
// This function stores a so called "database info file" in the database folder or
// checks if an existing "database info file" contains the correct engine.
// Otherwise the files in the database folder are not compatible.
func CheckDatabaseEngine(dbPath string, createDatabaseIfNotExists bool, dbEngine ...Engine) (Engine, error) {

	dbEngineSpecified := len(dbEngine) > 0 && dbEngine[0] != EngineAuto

	// check if the database exists and if it should be created
	dbExists, err := DatabaseExists(dbPath)
	if err != nil {
		return EngineUnknown, err
	}

	if !dbExists {
		if !createDatabaseIfNotExists {
			return EngineUnknown, fmt.Errorf("database not found (%s)", dbPath)
		}

		if createDatabaseIfNotExists && !dbEngineSpecified {
			return EngineUnknown, errors.New("the database engine must be specified if the database should be newly created")
		}
	}

	var targetEngine Engine

	// check if the database info file exists and if it should be created
	dbInfoFilePath := filepath.Join(dbPath, "dbinfo")
	_, err = os.Stat(dbInfoFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return EngineUnknown, fmt.Errorf("unable to check database info file (%s): %w", dbInfoFilePath, err)
		}

		if !dbEngineSpecified {
			return EngineUnknown, fmt.Errorf("database info file not found (%s)", dbInfoFilePath)
		}

		// if the dbInfo file does not exist and the dbEngine is given, create the dbInfo file.
		if err := storeDatabaseInfoToFile(dbInfoFilePath, dbEngine[0]); err != nil {
			return EngineUnknown, err
		}

		targetEngine = dbEngine[0]
	} else {
		dbEngineFromInfoFile, err := LoadDatabaseEngineFromFile(dbInfoFilePath)
		if err != nil {
			return EngineUnknown, err
		}

		// if the dbInfo file exists and the dbEngine is given, compare the engines.
		if dbEngineSpecified {

			if dbEngineFromInfoFile != dbEngine[0] {
				return EngineUnknown, fmt.Errorf(`database engine does not match the configuration: '%v' != '%v'

If you want to use another database engine, you can use the tool './hornet tool db-migration' to convert the current database.`, dbEngineFromInfoFile, dbEngine[0])
			}
		}

		targetEngine = dbEngineFromInfoFile
	}

	return targetEngine, nil
}

// LoadDatabaseEngineFromFile returns the engine from the "database info file".
func LoadDatabaseEngineFromFile(path string) (Engine, error) {

	var info databaseInfo

	if err := ioutils.ReadTOMLFromFile(path, &info); err != nil {
		return "", fmt.Errorf("unable to read database info file: %w", err)
	}

	return DatabaseEngineFromStringAllowed(info.Engine)
}

// storeDatabaseInfoToFile stores the used engine in a "database info file".
func storeDatabaseInfoToFile(filePath string, engine Engine) error {
	dirPath := filepath.Dir(filePath)

	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("could not create database dir '%s': %w", dirPath, err)
	}

	info := &databaseInfo{
		Engine: string(engine),
	}

	return ioutils.WriteTOMLToFile(filePath, info, 0660, "# auto-generated\n# !!! do not modify this file !!!")
}

type sqliteLogger struct {
	*logger.WrappedLogger
}

func newLogger(log *logger.Logger) *sqliteLogger {
	return &sqliteLogger{
		WrappedLogger: logger.NewWrappedLogger(log),
	}
}

func (l *sqliteLogger) Printf(t string, args ...interface{}) {
	l.LogWarnf(t, args...)
}

func DatabaseWithDefaultSettings(path string, createDatabaseIfNotExists bool, log *logger.Logger) (*gorm.DB, error) {

	targetEngine, err := CheckDatabaseEngine(path, createDatabaseIfNotExists, EngineSQLite)
	if err != nil {
		return nil, err
	}

	switch targetEngine {
	case EngineSQLite:
		dbFile := filepath.Join(path, "indexer.db")
		return gorm.Open(sqlite.Open(dbFile), &gorm.Config{
			Logger: gormLogger.New(newLogger(log), gormLogger.Config{
				SlowThreshold:             100 * time.Millisecond,
				LogLevel:                  gormLogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			}),
		})
	default:
		return nil, fmt.Errorf("unknown database engine: %s, supported engines: sqlite", targetEngine)
	}
}
