# amail

A Go command line tool that drives Apple Mail through macOS Automation. Read `GLOSSARY.md` for the project's vocabulary before naming anything.

## Agent skills

### Issue tracker

Issues, the wayfinder map and its decision tickets live in GitHub Issues on `ChiragAgg5k/amail`. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Working on mail safely

- Never send to third parties when testing. Use drafts, or `--send` only to the account's own address.
- Delete test messages and drafts afterwards, including autosaved copies in the Drafts mailbox.
- Leave mail and drafts that existed before the test alone.
