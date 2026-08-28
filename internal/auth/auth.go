package auth

import (
	"os"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/config"
	"github.com/truestandard/seo-cli/internal/output"
	"github.com/truestandard/seo-cli/internal/skills"
)

func LoginWithToken(cfg config.Config, noSkills bool) error {
	if cfg.Token == "" {
		return &api.APIError{Code: "usage", Message: "pass the key with --token seo_… or set SEO_API_KEY"}
	}
	identity, err := api.New(cfg).Whoami()
	if err != nil {
		return err
	}
	if err := config.SaveToken(cfg.Token, cfg.APIURL); err != nil {
		return err
	}
	output.Notice("Logged in to %s. Key saved at %s.", cfg.APIURL, config.CredentialsPath())
	installed := maybeInstallSkills(noSkills)
	output.Render(map[string]any{
		"status":           "authenticated",
		"api_url":          cfg.APIURL,
		"whoami":           identity,
		"skills_installed": installed,
	}, func() string { return "logged in to " + cfg.APIURL })
	return nil
}

func Logout() error {
	if err := config.DeleteCredentials(); err != nil {
		return err
	}
	output.Notice("Signed out. Key removed from %s.", config.CredentialsPath())
	output.Render(map[string]any{"status": "signed_out"}, func() string { return "signed out" })
	return nil
}

func maybeInstallSkills(noSkills bool) []skills.Result {
	if noSkills || !config.AutoInstallSkills() {
		return nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	results, err := skills.Install(dir)
	if err != nil {
		output.Notice("(skill install skipped: %v)", err)
		return nil
	}
	written := []skills.Result{}
	for _, r := range results {
		if r.Action == skills.ActionInstalled || r.Action == skills.ActionUpdated {
			output.Notice("Installed the seo skill for %s at %s.", r.Agent, r.Path)
			written = append(written, r)
		}
	}
	return written
}
