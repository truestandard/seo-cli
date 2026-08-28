package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultAPIURL = "https://truestandard.agency"

	EnvAPIURL    = "SEO_API_URL"
	EnvToken     = "SEO_API_KEY"
	EnvProject   = "SEO_PROJECT"
	EnvConfigDir = "SEO_CONFIG_DIR"
)

type Config struct {
	APIURL  string
	Token   string
	Project string
}

type credentials struct {
	Token  string `json:"token"`
	APIURL string `json:"api_url,omitempty"`
}

type fileConfig struct {
	Project       string `json:"project,omitempty"`
	InstallSkills *bool  `json:"install_skills,omitempty"`
}

func Resolve(flagAPIURL, flagToken, flagProject string) Config {
	creds, _ := loadCredentials()
	file, _ := loadFileConfig()
	return Config{
		APIURL:  strings.TrimRight(firstNonEmpty(flagAPIURL, os.Getenv(EnvAPIURL), creds.APIURL, DefaultAPIURL), "/"),
		Token:   firstNonEmpty(flagToken, os.Getenv(EnvToken), creds.Token),
		Project: firstNonEmpty(flagProject, os.Getenv(EnvProject), file.Project),
	}
}

func SaveToken(token, apiURL string) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(credentials{Token: token, APIURL: apiURL}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func DeleteCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func CredentialsPath() string {
	path, _ := credentialsPath()
	return path
}

func HasStoredToken() bool {
	creds, err := loadCredentials()
	return err == nil && creds.Token != ""
}

func SaveProject(slug string) error {
	file, _ := loadFileConfig()
	file.Project = slug
	return saveFileConfig(file)
}

func AutoInstallSkills() bool {
	file, err := loadFileConfig()
	if err != nil || file.InstallSkills == nil {
		return true
	}
	return *file.InstallSkills
}

func loadCredentials() (credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return credentials{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return credentials{}, nil
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return credentials{}, err
	}
	return c, nil
}

func loadFileConfig() (fileConfig, error) {
	path, err := configPath()
	if err != nil {
		return fileConfig{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, nil
	}
	var c fileConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return fileConfig{}, err
	}
	return c, nil
}

func saveFileConfig(c fileConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func Dir() (string, error) {
	if dir := os.Getenv(EnvConfigDir); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".seo"), nil
}

func credentialsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
