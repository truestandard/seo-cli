# Security

## Report a vulnerability

Email security@truestandard.ai. Include the version (`seo --version`), steps to reproduce, and what an attacker gains.

We reply within three business days and keep you posted until the fix ships. Please give us time to fix before you publish.

Do not open a public issue for a security bug.

## Supported versions

| Version | Supported |
|---|---|
| 0.1.x | yes |

Only the latest minor release gets fixes.

## What the CLI stores

`seo login` writes your API token to `~/.config/seo/config.json` with mode 0600. Set `SEO_TOKEN` in the environment instead if you prefer nothing on disk. The CLI sends the token as a Bearer header to the configured base URL and nowhere else.
