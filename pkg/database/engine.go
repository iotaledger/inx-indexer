package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/iotaledger/hive.go/core/ioutils"
	"github.com/iotaledger/hive.go/core/logger"
)

type Engine string

const (
	EngineUnknown  Engine = "unknown"
	EngineAuto     Engine = "auto"
	EngineSQLite   Engine = "sqlite"
	EnginePostgres Engine = "postgres"
)

type databaseInfo struct {
	Engine string `toml:"databaseEngine"`
}

type Params struct {
	Engine   Engine
	Path     string
	Host     string
	Port     uint
	Database string
	Username string
	Password string
}

// EngineFromString parses an engine from a string.
// Returns an error if the engine is unknown.
func EngineFromString(engineStr string) (Engine, error) {

	dbEngine := Engine(strings.ToLower(engineStr))

	//nolint:exhaustive // false positive
	switch dbEngine {
	case "":
		// no engine specified
		fallthrough
	case EngineAuto:
		return EngineAuto, nil
	case EngineSQLite:
		return EngineSQLite, nil
	case EnginePostgres:
		return EnginePostgres, nil
	default:
		return EngineUnknown, fmt.Errorf("unknown database engine: %s, supported engines: sqlite", dbEngine)
	}
}

// EngineAllowed checks if the database engine is allowed.
func EngineAllowed(dbEngine Engine, allowedEngines ...Engine) (Engine, error) {

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

	//nolint:exhaustive // false positive
	switch dbEngine {
	case EngineAuto:
	case EngineSQLite:
	case EnginePostgres:
	default:
		return "", fmt.Errorf("unknown database engine: %s, supported engines: sqlite", dbEngine)
	}

	return dbEngine, nil
}

// EngineFromStringAllowed parses an engine from a string and checks if the database engine is allowed.
func EngineFromStringAllowed(dbEngineStr string, allowedEngines ...Engine) (Engine, error) {

	dbEngine, err := EngineFromString(dbEngineStr)
	if err != nil {
		return EngineUnknown, err
	}

	return EngineAllowed(dbEngine, allowedEngines...)
}

// CheckEngine checks if the correct database engine is used.
// This function stores a so called "database info file" in the database folder or
// checks if an existing "database info file" contains the correct engine.
// Otherwise the files in the database folder are not compatible.
func CheckEngine(dbPath string, createDatabaseIfNotExists bool, dbEngine ...Engine) (Engine, error) {

	dbEngineSpecified := len(dbEngine) > 0 && dbEngine[0] != EngineAuto

	// check if the database exists and if it should be created
	dbExists, err := Exists(dbPath)
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
		dbEngineFromInfoFile, err := LoadEngineFromFile(dbInfoFilePath)
		if err != nil {
			return EngineUnknown, err
		}

		// if the dbInfo file exists and the dbEngine is given, compare the engines.
		if dbEngineSpecified {

			if dbEngineFromInfoFile != dbEngine[0] {
				//nolint:stylecheck,revive // this error message is shown to the user
				return EngineUnknown, fmt.Errorf(`database engine does not match the configuration: '%v' != '%v'

If you want to use another database engine, you can use the tool './hornet tool db-migration' to convert the current database.`, dbEngineFromInfoFile, dbEngine[0])
			}
		}

		targetEngine = dbEngineFromInfoFile
	}

	return targetEngine, nil
}

// LoadEngineFromFile returns the engine from the "database info file".
func LoadEngineFromFile(path string) (Engine, error) {

	var info databaseInfo

	if err := ioutils.ReadTOMLFromFile(path, &info); err != nil {
		return "", fmt.Errorf("unable to read database info file: %w", err)
	}

	return EngineFromStringAllowed(info.Engine)
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

func NewWithDefaultSettings(dbParams Params, createDatabaseIfNotExists bool, log *logger.Logger) (*gorm.DB, error) {

	targetEngine, err := CheckEngine(dbParams.Path, createDatabaseIfNotExists, dbParams.Engine)
	if err != nil {
		return nil, err
	}

	var dbDialector gorm.Dialector

	//nolint:exhaustive // false positive
	switch targetEngine {
	case EngineSQLite, EngineAuto:
		dbFile := filepath.Join(dbParams.Path, "indexer.db")
		dbDialector = sqlite.Open(dbFile)
	case EnginePostgres:
		dsn := fmt.Sprintf("host='%s' user='%s' password='%s' dbname='%s' port=%d", dbParams.Host, dbParams.Username, dbParams.Password, dbParams.Database, dbParams.Port)
		dbDialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unknown database engine: %s, supported engines: sqlite, postgres", targetEngine)
	}

	return gorm.Open(dbDialector, &gorm.Config{
		Logger: gormLogger.New(newLogger(log), gormLogger.Config{
			SlowThreshold:             100 * time.Millisecond,
			LogLevel:                  gormLogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		}),
	})
}
