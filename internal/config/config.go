package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"

func Read() (*Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	configFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	c := &Config{}
	decoder := json.NewDecoder(configFile)
	if err = decoder.Decode(c); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) SetUser(userName string) (bool, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return false, err
	}

	c.CurrentUserName = userName
	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return false, err
	}

	err = os.WriteFile(path, jsonData, 0o644)
	if err != nil {
		return false, err
	}

	return true, nil
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configFileName), nil
}
