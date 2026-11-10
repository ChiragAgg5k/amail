# Contributing to mailslot

Read [`GLOSSARY.md`](GLOSSARY.md) first and use its words in code, documentation and commit messages.

## The one rule

**Never test against someone's real mail.** That includes your own main account.

- Develop and test against the fixture backend. It answers in place of the Mail app.
- Run the end-to-end suite only against a dedicated throwaway account or local mailboxes that hold nothing you care about.
- Never send to third parties when testing. Use drafts, or `--send` only to the test account's own address.
- Delete test messages and drafts afterwards, including autosaved copies in Drafts. Leave mail that existed before the test alone.
- Do not paste real names, addresses, subjects or paths with a user name into code, fixtures, issues or recordings.

## Build

`mailslot` needs Go 1.25 or newer. It builds on any platform, though it only does something useful on macOS.

```sh
go build -o mailslot .
gofmt -l .        # must print nothing
go vet ./...
```

## Fixture backend

Setting `MAILSLOT_FIXTURE` to a directory of JSON files makes `mailslot` use that invented mail in place of the Mail app. Nothing reaches Mail.

```sh
MAILSLOT_FIXTURE=testdata/fixture ./mailslot doctor                 # the first row says "fixture backend"
MAILSLOT_FIXTURE=testdata/fixture ./mailslot --account Work list
MAILSLOT_FIXTURE=testdata/local ./mailslot mailboxes                # zero accounts, local mailboxes only
```

- `testdata/fixture` holds invented accounts (`Work`, `Personal`, `Old`), their mailboxes, local mailboxes and messages. `incoming.json` holds mail that arrives when `wait` checks for new mail.
- Changes are kept in memory and lost when the command ends. Set `MAILSLOT_FIXTURE_STATE` to a file outside the fixture directory to carry them from one command to the next. The fixture directory itself is never written. Use one state file per fixture directory.
- `MAILSLOT_FIXTURE_FAULT=permission` or `timeout` makes every call fail as a denied Automation permission (exit code 3) or an unanswered Mail (exit code 5).
- Fixture commands still queue on the same lock file as real ones (`~/Library/Caches/mailslot`), so they wait while another `mailslot` command is running.

## Tests

There are three layers.

### 1. Fixture tests

```sh
go test ./... -count=1
```

Command-level tests run every command against the fixture backend and compare table output, JSON output and exit codes. They never touch Mail and run in CI on every push.

### 2. Script compilation

The same `go test ./...` run assembles every embedded script in `internal/mail/script` with its prelude and compiles it with `osacompile` (`TestScriptsCompile` in `internal/mail`). This catches syntax errors without needing Mail. It is skipped off macOS.

Compiling proves a script parses, not that Mail does what it asks. Only the next layer proves behaviour.

### 3. End-to-end suite (opt-in)

> **Warning:** this suite drives the real Mail app and writes to the account you name. It marks, flags, moves, archives and deletes messages there, and on a real account it sends mail to that account's own address. Only ever point it at a dedicated throwaway account or at local mailboxes.

```sh
MAILSLOT_E2E_ACCOUNT=local go test -tags e2e ./internal/e2e/ -v -count=1 -timeout 15m
```

- `MAILSLOT_E2E_ACCOUNT` is the gate. Without it every test skips. Set it to the account's name in Mail, or to `local` for the On My Mac mailboxes.
- `MAILSLOT_E2E_MAILBOX` names the mailbox to take messages from (default `Inbox`).
- `MAILSLOT_E2E_TARGET` names a mailbox to move a message to and back from. Without it the move test skips.
- `MAILSLOT_E2E_MARKER` is a subject token that some message in the mailbox carries (default `mailslot-e2e-marker`), used by the search test.
- The suite refuses to start when `MAILSLOT_FIXTURE`, `MAILSLOT_READ_ONLY` or `MAILSLOT_NO_SEND` is set, and it runs with an empty configuration directory so your own defaults cannot change what is tested.
- Tests that need sending skip on `local`, which cannot send.

## How CI's `e2e-local` job works

`.github/workflows/ci.yml` has two jobs. `build` checks formatting, vets, builds and runs `go test ./...` on macOS. `e2e-local` then runs the end-to-end suite against a real Mail app on hosted macOS runners (`macos-26` and `macos-15`), which have no mail accounts:

1. It adds an Automation permission row for the runner agent to the runner's privacy database, before any request reaches Mail. A consent prompt nobody answers would otherwise leave a denial behind.
2. It launches Mail and checks that Mail answers a scripting request.
3. It creates local mailboxes for the tests to move messages into.
4. It builds an mbox file from `testdata/fixture` with `go run ./internal/fixture/mbox` and asks Mail to import it, then waits until the messages appear.
5. It runs the suite with `MAILSLOT_E2E_ACCOUNT=local`.

So the read and triage scripts run against a real Mail in CI, on invented mail. `send`, `reply`, `merge` and `wait` do not, because there is no account to send from.

Never run those workflow steps on your own Mac: they write to the privacy database and to Mail.

## Recordings

The GIFs in `docs/media` are made with [vhs](https://github.com/charmbracelet/vhs) from the tapes in `docs/media/tapes`. To regenerate them (needs `go`, `vhs` and `jq`):

```sh
docs/media/record.sh            # all of them
docs/media/record.sh triage     # just one
```

The script builds `mailslot` into `/tmp/mailslot-demo`, points `MAILSLOT_FIXTURE` at `testdata/fixture` through a link there, gives the recording its own `HOME` and configuration directory so that no personal path or setting can appear, and refuses to record unless `mailslot doctor` reports the fixture backend. Each tape starts from the untouched fixture; `MAILSLOT_FIXTURE_STATE` carries changes between the commands of one tape.

Run tapes through the script, not with `vhs` directly: a tape stops at its first line when the script's environment is missing, so it can never fall through to a `mailslot` on your `PATH` and your real mail.

After recording, look at the result before committing it. Keep each recording under about 20 seconds and 1.5 MB. Commit only the tapes and the GIFs, not frame dumps or screenshots.

The banner (`docs/media/banner.svg` and `banner-dark.svg`) is hand-written SVG. The two files differ only in colours; keep them in step.

## Commits and pull requests

- Conventional commits: `type(scope): subject` or `type: subject`, with types `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `style`, `perf`, `build`, `ci`, `revert`.
- Put the reason for the change in the body. The diff already says what changed.
- Run `gofmt`, `go vet ./...` and `go test ./... -count=1` before pushing.
- If a change alters what a command prints or accepts, update the README, `skills/mailslot/SKILL.md` and, when a recording shows it, the recording.

Issues, the project map and its decision tickets live in GitHub Issues; see [`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md).
