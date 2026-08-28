package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrecedenceFlagEnvFileDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvConfigDir, dir)
	t.Setenv(EnvAPIURL, "")
	t.Setenv(EnvToken, "")
	t.Setenv(EnvProject, "")

	cfg := Resolve("", "", "")
	if cfg.APIURL != DefaultAPIURL || cfg.Token != "" || cfg.Project != "" {
		t.Fatalf("defaults: %+v", cfg)
	}

	if err := SaveToken("seo_file", "http://file:3012/"); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject("file-project"); err != nil {
		t.Fatal(err)
	}
	cfg = Resolve("", "", "")
	if cfg.APIURL != "http://file:3012" || cfg.Token != "seo_file" || cfg.Project != "file-project" {
		t.Fatalf("file: %+v", cfg)
	}

	t.Setenv(EnvAPIURL, "http://env")
	t.Setenv(EnvToken, "seo_env")
	t.Setenv(EnvProject, "env-project")
	cfg = Resolve("", "", "")
	if cfg.APIURL != "http://env" || cfg.Token != "seo_env" || cfg.Project != "env-project" {
		t.Fatalf("env: %+v", cfg)
	}

	cfg = Resolve("http://flag", "seo_flag", "flag-project")
	if cfg.APIURL != "http://flag" || cfg.Token != "seo_flag" || cfg.Project != "flag-project" {
		t.Fatalf("flag: %+v", cfg)
	}
}

func TestCredentialsMode(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvConfigDir, dir)
	if err := SaveToken("seo_x", "http://x"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	if !HasStoredToken() {
		t.Fatal("expected a stored token")
	}
	if err := DeleteCredentials(); err != nil {
		t.Fatal(err)
	}
	if HasStoredToken() {
		t.Fatal("token still stored")
	}
	if err := DeleteCredentials(); err != nil {
		t.Fatalf("second delete must be a no-op: %v", err)
	}
}

func TestSaveProjectKeepsInstallSkills(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvConfigDir, dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"install_skills": false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveProject("p"); err != nil {
		t.Fatal(err)
	}
	if AutoInstallSkills() {
		t.Fatal("install_skills flag lost")
	}
	if Resolve("", "", "").Project != "p" {
		t.Fatal("project not saved")
	}
}
