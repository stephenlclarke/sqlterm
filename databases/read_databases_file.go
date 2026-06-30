package databases

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type FileDatabases struct {
	Databases []DatabaseCredentials `json:"databases"`
}

func ReadDatabasesJson() ([]DatabaseCredentials, error) {
	path, err := DatabasesFilePath()
	if err != nil {
		return nil, err
	}

	return ReadDatabasesJsonFromPath(path)
}

func DatabasesFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".config", "sqlterm", "databases.json"), nil
}

func ReadDatabasesJsonFromPath(path string) ([]DatabaseCredentials, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var databases FileDatabases
	if err := json.NewDecoder(jsonFile).Decode(&databases); err != nil {
		return nil, err
	}

	return databases.Databases, nil
}
