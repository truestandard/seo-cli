package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/truestandard/seo-cli/internal/api"
	"github.com/truestandard/seo-cli/internal/config"
	"github.com/truestandard/seo-cli/internal/output"
)

var (
	flagAPIURL  string
	flagToken   string
	flagProject string
	flagJSON    bool
	flagPretty  bool
	flagDryRun  bool
	flagQuiet   bool

	cfg config.Config
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "seo",
		Short: "TrueStandard Agency from the command line: keywords, ranks, AI answers, targets, spend",
		Long: "seo drives TrueStandard Agency, the SEO operator that runs the loop for every project.\n\n" +
			"Every command maps to one API call and prints JSON on stdout. Add --pretty for tables.\n" +
			"Paid commands take --dry-run (or `seo estimate <command>`) to price the call and spend nothing.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			output.Pretty = flagPretty && !flagJSON
			output.Quiet = flagQuiet
			cfg = config.Resolve(flagAPIURL, flagToken, flagProject)
		},
	}
	root.Version = version
	root.SetVersionTemplate("seo {{.Version}}\n")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageError(err.Error())
	})

	pf := root.PersistentFlags()
	pf.StringVar(&flagAPIURL, "api-url", "", "backend URL (default "+config.DefaultAPIURL+", env SEO_API_URL)")
	pf.StringVar(&flagToken, "token", "", "API key (overrides the saved key, env SEO_API_KEY)")
	pf.StringVar(&flagProject, "project", "", "project slug (overrides `seo use`, env SEO_PROJECT)")
	pf.BoolVar(&flagJSON, "json", false, "JSON on stdout (the default)")
	pf.BoolVar(&flagPretty, "pretty", false, "tables and key-value lines instead of JSON")
	pf.BoolVar(&flagDryRun, "dry-run", false, "paid commands: print the estimate and spend nothing")
	pf.BoolVar(&flagQuiet, "quiet", false, "no notices on stderr")

	root.AddCommand(
		authCmd(), loginCmd(), whoamiCmd(),
		projectsCmd(), projectCmd(), useCmd(), contextCmd(),
		keywordsCmd(), researchCmd(), serpCmd(), trackCmd(), ranksCmd(),
		promptsCmd(), aiCmd(), floorCmd(), backlinksCmd(), domainCmd(), mentionsCmd(), auditCmd(),
		gscCmd(), shipCmd(), experimentsCmd(), logCmd(), scoreboardCmd(), spendCmd(),
		estimateCmd(), mcpCmd(), skillsCmd(), versionCmd(),
	)
	return root
}

func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		if strings.HasPrefix(err.Error(), "unknown command") {
			err = usageError(err.Error())
		}
		output.DieErr(err)
	}
	os.Exit(output.ExitOK)
}

func usageError(message string) error {
	return &api.APIError{Code: "usage", Message: message, Status: 400}
}

func requireClient() (*api.Client, error) {
	if cfg.Token == "" {
		return nil, &api.APIError{
			Code:    "not_authenticated",
			Message: "no API key: run `seo auth login --token seo_…` or set SEO_API_KEY",
			Status:  401,
		}
	}
	return api.New(cfg), nil
}

func requireProject() (string, error) {
	if cfg.Project == "" {
		return "", usageError("no project selected: run `seo use <slug>`, pass --project <slug>, or set SEO_PROJECT")
	}
	return cfg.Project, nil
}

func projectClient() (*api.Client, string, error) {
	client, err := requireClient()
	if err != nil {
		return nil, "", err
	}
	slug, err := requireProject()
	if err != nil {
		return nil, "", err
	}
	return client, slug, nil
}
