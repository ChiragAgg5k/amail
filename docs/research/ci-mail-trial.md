# Trial: does Mail answer Automation on a hosted macOS runner with no account?

Run on 2026-11-10 for issue #22, following the sketch in [mail-on-ci-runners.md](https://github.com/ChiragAgg5k/amail/blob/research/mail-on-ci-runners/docs/research/mail-on-ci-runners.md). Everything below was observed in GitHub Actions runs of [`.github/workflows/mail-trial.yml`](../../.github/workflows/mail-trial.yml) on this branch, which is never merged.

## Answer

Yes, on all three hosted arm64 images (`macos-15`, `macos-26`, `xcode-27`), once one Automation row is added at job time. Without the row every Apple event to Mail hangs behind a consent prompt for `hosted-compute-agent` and never returns.

With one `kTCCServiceAppleEvents` row for client `/opt/hca/hosted-compute-agent` and target `com.apple.mail` in the per-user TCC database, Mail with zero accounts answers `osascript`, creates an "On My Mac" mailbox, imports an mbox file and returns the messages in it. Mail does show an add-account sheet, but the sheet does not block scripting.

The recipe gave identical results in 3 out of 3 executions of the final workflow (18 seeded jobs, no failure, no retry of the TCC insert). The one failure seen in the whole trial had a known cause: sending an Apple event before seeding, which leaves a deny row.

`amail` itself can use almost none of this today: `amail accounts --json` prints `[]`, and every other command stops at "Mail has no accounts".

## Results per image

Final workflow, [run 42806220600](https://github.com/ChiragAgg5k/amail/actions/runs/42806220600), attempt 1. Attempts 2 and 3 produced the same exit code for all 132 recorded probes.

| Step | `macos-15` (macOS 15.8.2, image 20261104.0514.1, Mail build 3826.700.81.1.11) | `macos-26` (macOS 26.7.2, image 20261102.0473.1, Mail build 3864.700.51.1.4) | `xcode-27` (macOS 27.0.1, image 20261104.0337.1, Mail build 3901.100.1.1.12) |
| --- | --- | --- | --- |
| SIP | disabled | disabled | disabled |
| Responsible process for TCC | `/opt/hca/hosted-compute-agent` | `/opt/hca/hosted-compute-agent` | `/opt/hca/hosted-compute-agent` |
| Existing Automation row for Mail | none | none | none |
| No grant: `Application('Mail').version()` | `16.0`, exit 0, but Mail is not launched (answered from the bundle) | same | same |
| No grant: `accounts.length` | killed by the 60 s timeout (exit 142), consent prompt on screen | same | same |
| No grant: TCC row left behind | deny row for the agent (`auth_value` 0, `auth_reason` 9) | same | same |
| Insert the agent row into the user TCC.db | exit 0, first try | exit 0, first try | exit 0, first try (database is under `/private/var/containers/Data/ProtectedSystem/<uuid>/...`) |
| `open -g -a Mail` | exit 0, window "Inbox (Read Only)" with a sheet "Choose a Mail account provider..." | exit 0, window "Inbox (Read Only)" with a sheet "Add Account" | exit 0, same as `macos-26` |
| `accounts.length` | `0` | `0` | `0` |
| `mailboxes.length` | `0` | `0` | `0` |
| `make new mailbox` named `amail-ci-probe` (JXA `Mail.mailboxes.push(Mail.Mailbox({name}))`) | exit 0, mailbox listed afterwards | same | same |
| `import Mail mailbox` of a two-message mbox | exit 0, mailboxes `Import` and `Import/probe` appear | same | same |
| Message count in the imported mailbox | 2 | 2 | 2 |
| Read one subject, sender and body | `amail ci probe two`, `Carol Example <carol@example.com>`, `Second invented message.` | same | same |
| `go build` | exit 0 | exit 0 | exit 0 |
| `amail accounts --json` | `[]`, exit 0 | `[]`, exit 0 | `[]`, exit 0 |
| `amail mailboxes --json` | exit 1, `amail: Mail has no accounts; add one in System Settings → Internet Accounts` | same | same |
| Escape on the sheet (System Events) | Mail quits; the next Apple event relaunches it with the sheet back and the mailboxes and messages intact | same | same |

Seeding all six clients the image lists for Finder (`seed all`) and seeding only the agent row (`seed agent`) gave the same results on every image, so the agent row is the one that matters. Rows for `/bin/bash` and `/usr/bin/osascript` are not needed.

## Runs

| Run | What it tested | Outcome |
| --- | --- | --- |
| [42803382139](https://github.com/ChiragAgg5k/amail/actions/runs/42803382139) | Seed all clients, then probe. The baseline before seeding was only `version()`. | All probes passed on all three images. The baseline proved nothing because `version()` never reaches Mail. |
| [42803908547](https://github.com/ChiragAgg5k/amail/actions/runs/42803908547) | A real Apple event before seeding, plus jobs that never seed. | Unseeded events time out everywhere (the never-seeded jobs ran into the 15 minute job limit and were cancelled, as expected). Seeding after the timed-out prompt worked on `macos-15` and `macos-26` but not on `xcode-27`, where the agent's row still read `auth_value` 0 right after the insert and every probe timed out. |
| [42806220600](https://github.com/ChiragAgg5k/amail/actions/runs/42806220600) attempt 1 | Final design: `seed none` only records the prompt, `seed agent` and `seed all` seed before any Apple event. | 9 jobs, all as in the table above. |
| [42806220600 attempt 2](https://github.com/ChiragAgg5k/amail/actions/runs/42806220600/attempts/2) | Repeat | Identical exit codes for all 132 probes. |
| [42806220600 attempt 3](https://github.com/ChiragAgg5k/amail/actions/runs/42806220600/attempts/3) | Repeat | Identical exit codes for all 132 probes. |

Five executions in total (three pushes and two re-runs). Each job's summary page has the full table of exit codes, stdout and stderr; screenshots and the filtered `tccd` log are in the `trial-<image>-seed-<variant>` artifacts.

## What the screenshots showed

- Without a grant: a system dialog reading "hosted-compute-agent" wants access to control "Mail", with Don't Allow and Allow buttons, over an otherwise empty desktop. Nothing dismisses it, and `osascript` blocks until it is killed.
- After seeding and `open -g -a Mail`: Mail's main window titled "Inbox (Read Only)" with an account sheet on top of it. On `macos-26` and `xcode-27` the sheet is "Add Account" with an email field and a Cancel button; on `macos-15` it is "Choose a Mail account provider..." with a Quit button.
- After the probes: the same sheet, and behind it the sidebar shows an "On My Mac" section containing `amail-ci-probe` and `Import` with an unread count of 2. The sheet stays up the whole time and scripting works regardless.
- After pressing Escape: an empty desktop. Mail has quit.

## Flakiness

- Final workflow: 3 executions x 6 seeded jobs = 18 seeded jobs, 18 successes. 3 x 3 unseeded jobs all timed out the same way.
- The TCC insert never failed and never needed its retry loop in any of the 5 executions (24 seeded jobs across all runs). The "database is locked" failures reported by others did not occur here.
- The only seeded failure (run 42803908547, `xcode-27`) followed an unseeded Apple event in the same job. On every image a timed-out prompt leaves a deny row for the agent; on `xcode-27` the later insert did not take effect.
- Limits of this evidence: all executions ran within about an hour on the same image versions. It says nothing about what happens when an image is rebuilt, when the agent's path changes, or when macOS 27 leaves preview.

## Minimal working recipe

Run this before anything sends an Apple event to Mail. It copies the image's own Finder row for the runner agent, so it follows whatever columns the `access` table has on that image.

```bash
DB="$HOME/Library/Application Support/com.apple.TCC/TCC.db"
if [ "$(sw_vers -productVersion | cut -d. -f1)" -ge 27 ]; then
  DB=$(sudo lsof -c tccd -Fn | sed -n 's/^n//p' | grep '/ProtectedSystem/.*/com\.apple\.TCC/TCC\.db$' | sort -u | head -1)
fi
columns=$(sudo sqlite3 "$DB" "select group_concat(case name when 'indirect_object_identifier' then '''com.apple.mail''' when 'last_modified' then 'cast(strftime(''%s'', ''now'') as integer)' else '\"' || name || '\"' end, ', ') from pragma_table_info('access')")
sudo sqlite3 -cmd '.timeout 10000' "$DB" "insert or replace into access select $columns from access where service = 'kTCCServiceAppleEvents' and indirect_object_identifier = 'com.apple.finder' and client = '/opt/hca/hosted-compute-agent'"
sudo killall tccd
open -g -a Mail
```

Then:

- Wrap every `osascript` call in a hard timeout (`perl -e 'alarm shift; exec @ARGV' 60 osascript ...`). A missing grant shows up as a hang, not as an error.
- Do not use `Application('Mail').version()` as a health check. It succeeds without a grant and without launching Mail. Use `accounts.length`.
- Leave the account sheet alone. Dismissing it quits Mail.
- Fixtures: `Mail.mailboxes.push(Mail.Mailbox({name: '...'}))` creates a local mailbox. `Mail.importMailMailbox({at: Path('/path/probe.mbox')})`, where `probe.mbox` is a folder holding a standard mbox file named `mbox`, creates `Import/probe` with the messages in it. The import returned within the 15 seconds the workflow waited before counting.
- `Mail.mailboxes()` lists a nested mailbox twice, once under its leaf name (`probe`) and once inside its parent (`Import` then `probe`).

The trial did not kill `tccd` separately from seeding, so whether `sudo killall tccd` is required is untested; it was always run.

## What it means for the test architecture

Hosted CI end-to-end is feasible but fragile.

- Feasible: the mechanism works on every current arm64 image, costs about two minutes per job, needs no secrets and no account, and local mailboxes with imported messages give list, read and search something real to return.
- Fragile: it depends on SIP staying disabled on the images, on direct writes to TCC.db (unsupported by Apple, and the database already moved on macOS 27), and on the agent's path staying `/opt/hca/hosted-compute-agent` (it has changed once before). A break will appear as timeouts, so an end-to-end job should start with a single guarded `accounts.length` probe and report "grant not effective" clearly instead of failing 20 tests by timeout.
- Not yet useful for `amail`: every command except `accounts` resolves a named account first, so on a runner only `amail accounts` and the no-account error path can be exercised. End-to-end coverage of list, read and search needs `amail` to address top-level "On My Mac" mailboxes (they are `mailbox` elements of the application, not of an account). Sending, replying and anything account-specific stays out of reach on a hosted runner.
- Suggested shape: keep unit tests hermetic as today; add an opt-in end-to-end job on `macos-26` (and `macos-15` if wanted) that seeds the agent row, imports an mbox fixture and runs the read-side commands once they can target local mailboxes; treat `xcode-27` as informational while it is a preview; keep a local-only suite for the account-backed paths.
