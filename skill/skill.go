package skill

import _ "embed"

const Version = "1"

//go:embed body.md
var body string

const description = "Drive TrueStandard Agency from the `seo` CLI: projects, keywords, " +
	"rank checks, live search results, AI answer scans, win checks, backlinks, " +
	"Search Console imports, shipped pages and spend. Use when asked to track " +
	"rankings, research keywords, check if a term is winnable, log a shipped page, " +
	"or read the weekly scoreboard. TRIGGERS: seo, rank check, keyword research, " +
	"AI visibility, scoreboard, ship a page, spend on data."

const (
	BeginPrefix = "<!-- BEGIN seo-cli"
	BeginMarker = BeginPrefix + " v" + Version + " (managed by `seo skills`, do not edit by hand) -->"
	EndMarker   = "<!-- END seo-cli (managed) -->"
)

func ClaudeSkill() string {
	return "---\n" +
		"name: seo\n" +
		"description: " + description + "\n" +
		"---\n\n" +
		"# seo CLI\n\n" +
		body
}

func AgentsBlock() string {
	return BeginMarker + "\n\n" +
		"## seo CLI\n\n" +
		body +
		"\n" + EndMarker + "\n"
}
