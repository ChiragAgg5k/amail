# Can a GitHub-hosted macOS runner script Mail without a mail account?

Researched 2026-11-10 for issue #2. Reading only: no CI run was triggered, so nothing below was observed on a runner by this project itself.

## Answer

Not out of the box, and nobody has publicly shown it working for Mail. The runner images pre-approve Automation (TCC `kTCCServiceAppleEvents`) only for Finder, System Events, Safari and Terminal as targets, so the first `osascript` call to Mail would hit an unanswerable consent prompt and time out.

It is probably fixable at job time: every current image (`macos-15`, `macos-26`, `macos-26-intel`, `xcode-27`) runs with SIP disabled and passwordless `sudo`, and several public projects seed their own TCC rows with `sqlite3` in a workflow step and then script Notes, Contacts, Reminders, Calendar or Safari successfully. A row for target `com.apple.mail` is the same technique, but it is unsupported by Apple, has broken across image updates before, and gets harder on macOS 27.

Two things remain unknown and only a trial run can settle them: whether Mail with zero accounts answers Apple events or sits behind its add-account window, and whether a message can be put into an "On My Mac" mailbox without any account. Separately, `amail` today resolves every command through a named account, so an account-less runner could exercise almost none of it without code changes.

If the trial fails, the realistic fallback is a local-only end-to-end suite (or a self-hosted Mac with a signed-in test account); hosted Cirrus CI no longer exists.

## Evidence

### Which macOS labels exist today

Verified from the runner-images README and the GitHub-hosted runners reference:

- `macos-latest`, `macos-26`, `macos-26-xlarge` are macOS 26 arm64; `macos-26-intel`, `macos-26-large`, `macos-latest-large` are macOS 26 x64; `macos-15`, `macos-15-xlarge` are macOS 15 arm64; `macos-15-intel`, `macos-15-large` are macOS 15 x64; `xcode-27`, `xcode-27-xlarge` are a macOS 27 arm64 public preview. Source: https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/README.md and https://docs.github.com/en/actions/reference/runners/github-hosted-runners
- macOS 14 images were retired from GitHub Actions on 2026-11-02. Source: https://github.com/actions/runner-images/issues/15226
- The Xcode 27 image is still a preview. Source: https://github.com/actions/runner-images/issues/14404
- Standard arm64 macOS runners have 3 cores and 7 GB of memory, and "Nested-virtualization is not supported due to the limitation of Apple's Virtualization Framework". Source: https://docs.github.com/en/actions/reference/runners/github-hosted-runners#limitations-for-arm64-macos-runners

### What the images pre-approve for Automation

Verified by reading the image build scripts at commit `741e4eb`:

- The macOS 15 and 26 templates (x64 and arm64) run `configure-tccdb-macos.sh`; the Xcode 27 template runs `configure-tccdb-macos-27.sh`. Source: https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/templates/macOS-26.arm64.anka.pkr.hcl#L178 and https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/templates/xcode-27.arm64.anka.pkr.hcl#L178
- Both scripts insert rows straight into the system and per-user TCC databases with `sqlite3` (`INSERT OR IGNORE INTO access VALUES(...)`). Source: https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/scripts/build/configure-tccdb-macos.sh and https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/scripts/helpers/utils.sh#L170-L182
- The `kTCCServiceAppleEvents` rows cover these client and target pairs, and no others:

| Client (sender) | Targets pre-approved |
| --- | --- |
| `/bin/bash` | `com.apple.systemevents`, `com.apple.finder`, `com.apple.Safari` |
| `/usr/bin/osascript` | `com.apple.systemevents`, `com.apple.finder`, `com.apple.Safari` |
| `/opt/hca/hosted-compute-agent` (the current runner agent) | `com.apple.finder`, `com.apple.systemevents`, `com.apple.Safari` |
| `/usr/local/opt/runner/provisioner/provisioner` (the older agent) | `com.apple.finder`, `com.apple.systemevents` |
| `/opt/hca/start_hca.sh`, `/usr/local/opt/runner/runprovisioner.sh` | `com.apple.systemevents` |
| `/usr/libexec/sshd-keygen-wrapper`, the Anka guest agent, `com.apple.Terminal` | `com.apple.finder`, `com.apple.systemevents`, `com.apple.Terminal` |

- There is no AppleEvents row with `com.apple.mail` (or Notes, Reminders, Calendar, Contacts) as the target in either script. The only mention of Mail is an unrelated `kTCCServiceUbiquity` (iCloud) row for client `com.apple.mail`.
- The same scripts also grant `/bin/bash` Accessibility, Screen Capture and Full Disk Access (`kTCCServiceSystemPolicyAllFiles`), and grant `/usr/bin/osascript` Accessibility. The System Events grant means a job can already inspect windows and press keys through `osascript`, which is how other projects dismiss first-run dialogs.
- The Safari rows were added only after users reported the prompt `"hosted-compute-agent" wants access to control "Safari"` blocking their jobs. That report also shows which process TCC holds responsible on a runner: the agent, not `osascript`. Source: https://github.com/actions/runner-images/issues/12616 and https://github.com/actions/runner-images/pull/12510
- A third-party workflow that read `tccd`'s own log on a `macos-26` image records the same attribution: `responsible_path=/opt/hca/hosted-compute-agent`. Source: https://github.com/nikgraf/calendar/blob/156e0b19125c58a2efe38fea0d9677ee86014c74/apps/desktop/e2e/ci/grant-reminders-tcc.sh

Inference: an un-seeded `osascript -l JavaScript` call to `Application('Mail')` on any current image raises a consent dialog nobody can click and ends in `AppleEvent timed out (-1712)`, exactly as Finder did before the images were seeded (https://github.com/actions/runner-images/issues/553, https://github.com/actions/runner-images/issues/7531).

### SIP, and whether a job can add TCC rows itself

Verified:

- SIP is disabled on all four current image families. A public workflow that runs `csrutil status` printed `System Integrity Protection status: disabled.` on 2026-11-06 for `macos-15` (image `macos-15-arm64` 20261104.0514.1), `macos-26` (image `macos-26-arm64` 20261102.0473.1), `macos-26-intel` (image `macos-26` 20260824.0517.1) and `xcode-27` (image `xcode-27-arm64` 20261104.0337.1). Source: https://github.com/GPSBabel/gpsbabel/actions/runs/42304091515/job/128192176950 , https://github.com/GPSBabel/gpsbabel/actions/runs/42304091515/job/128192176970 , https://github.com/GPSBabel/gpsbabel/actions/runs/42304091515/job/128192177068 , https://github.com/GPSBabel/gpsbabel/actions/runs/42304091515/job/128192176928
- This has been the stated design since macOS 13: maintainers said SIP may stay disabled "due to some architectural limitations" and that a workflow cannot toggle it. Source: https://github.com/actions/runner-images/issues/8162 and https://github.com/actions/runner-images/issues/9091
- The image build itself assumes SIP is off (it writes under `/private/var/db/Accessibility` and unloads system launch daemons "requires SIP to be disabled"). Source: https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/scripts/build/configure-machine.sh and https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/scripts/build/configure-system.sh
- The image sets `%admin ALL = (ALL) NOPASSWD: ALL`, so `sudo sqlite3` works without a password. Source: same `configure-machine.sh`.
- When SIP was still enabled (macOS 10.15 era) maintainers said the TCC problem "can't be resolved in runtime". Source: https://github.com/actions/runner-images/issues/553
- Maintainers have themselves recommended job-time `sudo sqlite3 ... INSERT OR REPLACE INTO access VALUES('kTCCServiceAppleEvents', ...)` as the workaround, including for a non-pre-approved Apple app (Notes), and posted a passing run. Source: https://github.com/actions/runner-images/issues/8214 and https://github.com/actions/runner-images/issues/7531

Caveats, also verified:

- Job-time writes fail intermittently. A report from August 2026 shows `sqlite3` inserts into the user TCC.db failing on `macos-15`, and a December 2025 comment shows the same on `macos-14`. Source: https://github.com/actions/runner-images/issues/14566 and https://github.com/actions/runner-images/issues/12616
- One large project attributes such failures to `tccd` holding a lock and wraps the insert in a retry loop with a busy timeout. Source: https://github.com/electron/electron/blob/fcb3c2a65b7f732e351da8e2fe38787ed490008e/.github/workflows/pipeline-segment-electron-test.yml#L132-L179
- The `access` table's columns have changed between macOS versions (10.15 to 11, and again in 14), breaking positional inserts each time. Source: https://github.com/actions/runner-images/issues/553 (later comments) and the header of https://github.com/nikgraf/calendar/blob/156e0b19125c58a2efe38fea0d9677ee86014c74/apps/desktop/e2e/ci/grant-reminders-tcc.sh
- The responsible process changed from the provisioner to `hosted-compute-agent` in mid-2025 and silently invalidated existing seeds; grants have to name every plausible client. Source: https://github.com/actions/runner-images/issues/12616
- macOS 27 moves the per-user database into `/private/var/containers/Data/ProtectedSystem/<uuid>/...`, and Apple's release notes say "Apps can no longer access the local TCC database directly." The image script and third-party scripts still reach it with `sudo` on the SIP-disabled preview image. Source: https://developer.apple.com/documentation/macos-release-notes/macos-27-release-notes#TCC , https://github.com/actions/runner-images/blob/741e4eb8ef5f51698d730ac16ede65cadb067522/images/macos/scripts/build/configure-tccdb-macos-27.sh , https://github.com/nikgraf/calendar/blob/156e0b19125c58a2efe38fea0d9677ee86014c74/apps/desktop/e2e/ci/tcc-user-db.sh

### Does Mail start without an account prompt

Not verified by anything found. What the sources do and do not say:

- Apple's Mail guide says "Before you can send email, you need to add your email account" and illustrates Mail showing an account dialog, but does not document what Mail does when it is launched with no account, or whether it answers Apple events in that state. Source: https://support.apple.com/guide/mail/get-started-mlhl5094a9f2/mac
- Apple documents "On My Mac" mailboxes as local-only mailboxes, without saying whether they can be created before any account exists. Source: https://support.apple.com/guide/mail/create-or-delete-mailboxes-mlhlp1021/mac
- No runner-images issue mentions Mail. Searches of that repository's issues for `Mail.app`, `TCC AppleEvents`, and `osascript -1743` return nothing relevant.
- Other Apple apps do stall behind first-run windows on runners. Photos did not answer Apple events until a script activated it and pressed Return through System Events. Source: https://github.com/actions/runner-images/issues/553 (September 2021 comments) and https://github.com/japlscript/japlphoto/blob/8adfe022545db94f91c26673fe625cb7d991f944/.github/workflows/build-and-test.yml
- One Mail automation project deliberately keeps Mail out of CI: its workflow comment says its tests make "real osascript subprocess calls ... never touching Application("Mail")". Source: https://github.com/ErnestoCobos/cobos-apple-mail-mcp/blob/7a0946a97785df9ec0584fcadbff7b6d69a20fc4/.github/workflows/ci.yml

Inference: Mail is part of the sealed system volume, so it is present on every image; whether it is scriptable with zero accounts is unknown. The runner does have a logged-in GUI session (the image configures auto-login and its build scripts drive Finder and System Events with `osascript`), so "headless" is not the obstacle; the account sheet is.

### Does anyone already do this in GitHub Actions

Verified by GitHub code search on 2026-11-10 (searches are best-effort and index only default branches):

- Mail: no workflow found. Queries `kTCCServiceAppleEvents com.apple.mail path:.github/workflows`, `TCC.db com.apple.mail path:.github`, `"open -a Mail" path:.github/workflows` and `osascript "application \"Mail\"" path:.github/workflows` all returned zero workflow hits. The only Mail-related workflow found is the one above that avoids Mail.
- Notes on a hosted runner (`macos-13`, now retired), seeding an AppleEvents row for `com.apple.Notes`: https://github.com/TomasHubelbauer/apple-notes-database/blob/a543131662959a3b84e6797008b565a91a85b0a5/.github/workflows/playwright.yml (no recent runs are retained, so current status is unknown).
- Contacts, Photos, Finder, Music and Spotify through AppleScript, each seeding an AppleEvents row first: https://github.com/japlscript/japlcontact/blob/562c52396df32c7d1c524f64717175db3af886ac/.github/workflows/build-and-test.yml , https://github.com/japlscript/japlphoto/blob/8adfe022545db94f91c26673fe625cb7d991f944/.github/workflows/build-and-test.yml , https://github.com/japlscript/obstmusic/blob/14077260a6d342f619ff239ec64f07f8e8e751c2/.github/workflows/build-and-test.yml (most of these still name the long-gone `macos-10.15` label; no recent runs are retained).
- Reminders and Calendar through EventKit on `xcode-27`, seeding `kTCCServiceReminders` and `kTCCServiceCalendar`, then probing that the grant took effect. This is not Apple events, but it is the most current proof that job-time seeding works, including on macOS 27; the job passed on 2026-11-09. Source: https://github.com/nikgraf/calendar/blob/156e0b19125c58a2efe38fea0d9677ee86014c74/.github/workflows/ci.yml and https://github.com/nikgraf/calendar/actions/runs/42655523557/job/129706676945
- Contacts privacy answered in advance on `macos-latest` for the agent, provisioner, `bash` and `osascript`, then the Shortcuts app is driven; passing on 2026-11-07. Source: https://github.com/frontboat/shortcutkit/blob/8f613f0d7f44b2915750d2c735cdca55285e6457/.github/workflows/ci.yml and https://github.com/frontboat/shortcutkit/actions/runs/42382623466/job/128433267944
- Safari controlled by `osascript` on `macos-latest`, seeding AppleEvents rows for `bash`, `osascript` and the discovered `hosted-compute-agent` path, with a background script that clicks any remaining prompt. Source: https://github.com/scottatron/scottatron.com/blob/b3b3b0b146ef8cd9a57cd77368507ac891994104/.github/workflows/export-pdf.yml (last retained run succeeded after four failures the same day).
- An app-level example that seeds AppleEvents, Accessibility, Calendar and more for its own bundle on `xcode-27`: https://github.com/plyght/foyer/blob/e74d2be1e2097f79ee1a8b76b94840c71a01ae01/.github/workflows/ui-capture.yml

### A constraint in this repository

Verified in this repository's scripts: every command resolves a named account first (`Mail.accounts.byName(name)` in `internal/mail/script/prelude.js`, used by `mailboxes.js` and, through `findMailbox`, by the read, list and search paths), and the README states "Requires macOS with at least one account set up in Mail." Mail's scripting dictionary exposes local mailboxes as `mailbox` elements of the application itself, not of an account (checked with `sdef /System/Applications/Mail.app` on macOS 26.6.2).

Consequence: even if Mail answers on a runner, an account-less run could only exercise `amail accounts` (expecting an empty list) and the error paths, unless `amail` grows a way to address top-level "On My Mac" mailboxes.

## Alternatives if the trial fails

- Local-only end-to-end suite. Needs nothing new: a developer Mac with Mail signed in and the terminal already allowed under Automation. Gate the tests behind a build tag or an environment variable so `go test ./...` in CI stays hermetic, and keep CI to unit tests that feed recorded `osascript` output to the Go parsing code. This is what the one comparable Mail project found does.
- Self-hosted Mac runner. Needs a Mac that stays logged in to a GUI session, Mail signed in to a throwaway account (or the account-less state, if the trial shows it works), and a one-time click on the Automation prompt for the process TCC holds responsible. The runner's macOS service is a per-user LaunchAgent with `ProcessType Interactive` and `SessionCreate`, so it runs inside the user session, but the responsible process becomes the runner binary rather than a terminal, so the grant must be given to that. Source: https://github.com/actions/runner/blob/main/src/Misc/layoutbin/darwin.svc.sh.template , https://github.com/actions/runner/blob/main/src/Misc/layoutbin/actions.runner.plist.template , https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/configure-the-application . A self-hosted runner on a public repository also runs untrusted pull-request code on that Mac, with access to whatever Mail holds.
- Tart VM on owned Apple Silicon hardware. Tart is still maintained, and the Cirrus image templates include a recovery-boot `csrutil disable` template and a TCC seeding script, so a VM with SIP off and a Mail grant baked in is buildable. Source: https://github.com/cirruslabs/macos-image-templates/blob/main/templates/disable-sip.pkr.hcl and https://github.com/cirruslabs/macos-image-templates/blob/main/scripts/update-tcc-database.sh . It needs a host Mac and a self-hosted runner that starts the VM per job. It cannot run inside a GitHub-hosted arm64 runner: nested virtualization is unsupported there, and Tart's own FAQ says nesting works only for Linux guests. Source: https://tart.run/faq/ . One public workflow that tries Tart on `macos-15` has only failed runs on record (their logs have expired, so the cause is unconfirmed): https://github.com/aidanns/agent-auth/blob/3c0d0a02618b7eb127872659ca12f58f1d5c4bc3/.github/workflows/test-system-macos-tart.yml
- Hosted Cirrus. Not available: Cirrus CI shut down on 2026-06-01 and Cirrus Runners stopped accepting new customers. Source: https://cirruslabs.org/
- Other hosted macOS CI was not investigated.

## Open unknowns that need a trial run

1. Does an un-seeded `osascript` call to Mail fail with a timeout (-1712), a denial (-1743), or succeed?
2. After seeding `kTCCServiceAppleEvents` rows with target `com.apple.mail`, does the call get through, and which client row is the one that matters (the agent, `bash`, or `osascript`)?
3. With zero accounts, does Mail answer `accounts()` and `mailboxes()`, or does it block behind an add-account window? If it blocks, can System Events dismiss that window and leave Mail scriptable?
4. Can a local mailbox be created by script with no account, and can a message be placed in it (for example by saving an outgoing message, or by opening an `.eml` file) so that list, read and search have something to return?
5. Does any of this differ between `macos-15`, `macos-26` and `xcode-27`?
6. How often does the TCC insert fail because the database is locked?

### What the trial workflow would contain

An untested sketch. It is `workflow_dispatch` only so it never runs on pushes, and every probing step is allowed to fail so one run answers all the questions.

```yaml
name: mail-scripting-trial
on: workflow_dispatch
permissions:
  contents: read
jobs:
  probe:
    strategy:
      fail-fast: false
      matrix:
        os: [macos-15, macos-26, xcode-27]
    runs-on: ${{ matrix.os }}
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Environment
        run: |
          sw_vers
          csrutil status
          pgrep -lf hosted-compute-agent || true
          DB="$HOME/Library/Application Support/com.apple.TCC/TCC.db"
          sudo test -f "$DB" || DB=$(sudo lsof -a -u "$(id -u)" -c tccd -Fn | sed -n 's/^n//p' | grep 'ProtectedSystem/.*/com.apple.TCC/TCC.db$' | sort -u | head -1)
          echo "TCC_DB=$DB" >> "$GITHUB_ENV"
          sudo sqlite3 "$DB" ".schema access"
          sudo sqlite3 "$DB" "select client, indirect_object_identifier, auth_value from access where service = 'kTCCServiceAppleEvents'"

      - name: Baseline without a grant (question 1)
        continue-on-error: true
        run: |
          osascript -l JavaScript -e 'const Mail = Application("Mail"); JSON.stringify(Mail.accounts.name())' &
          pid=$!
          sleep 30
          screencapture -x baseline.png || true
          kill "$pid" 2>/dev/null && echo "still blocked after 30s" || wait "$pid"
          osascript -e 'tell application "Mail" to quit' >/dev/null 2>&1 &
          pkill -x Mail || true

      - name: Seed Automation grants for Mail (question 2)
        run: |
          now=$(date +%s)
          for client in /opt/hca/hosted-compute-agent /usr/local/opt/runner/provisioner/provisioner /bin/bash /usr/bin/osascript; do
            for attempt in 1 2 3 4 5; do
              sudo sqlite3 -cmd '.timeout 10000' "$TCC_DB" "INSERT OR REPLACE INTO access (service, client, client_type, auth_value, auth_reason, auth_version, csreq, policy_id, indirect_object_identifier_type, indirect_object_identifier, indirect_object_code_identity, flags, last_modified) VALUES ('kTCCServiceAppleEvents', '$client', 1, 2, 0, 1, NULL, NULL, 0, 'com.apple.mail', NULL, NULL, $now)" && break
              echo "insert failed for $client (attempt $attempt)"; sleep $((attempt * 2))
            done
          done
          sudo killall tccd || true
          sudo sqlite3 "$TCC_DB" "select client, auth_value from access where service = 'kTCCServiceAppleEvents' and indirect_object_identifier = 'com.apple.mail'"

      - name: Launch Mail and record what is on screen (question 3)
        continue-on-error: true
        run: |
          open -a Mail
          sleep 20
          screencapture -x launched.png || true
          osascript -e 'tell application "System Events" to get {name, subrole} of every window of process "Mail"'
          osascript -e 'tell application "System Events" to get name of every sheet of every window of process "Mail"'

      - name: Read with no account (question 3)
        continue-on-error: true
        timeout-minutes: 2
        run: |
          osascript -l JavaScript -e 'const Mail = Application("Mail"); JSON.stringify({accounts: Mail.accounts.name(), mailboxes: Mail.mailboxes.name(), inbox: Mail.inbox.messages.length})'

      - name: Create a local mailbox and a message (question 4)
        continue-on-error: true
        timeout-minutes: 3
        run: |
          osascript -l JavaScript -e 'const Mail = Application("Mail"); Mail.mailboxes.push(Mail.Mailbox({name: "amail-ci"})); JSON.stringify(Mail.mailboxes.name())'
          printf 'From: a@example.com\nTo: b@example.com\nSubject: amail ci probe\nDate: Tue, 10 Nov 2026 10:00:00 +0000\nMessage-ID: <probe@example.com>\n\nbody\n' > probe.eml
          open -a Mail probe.eml
          sleep 10
          screencapture -x eml.png || true
          osascript -l JavaScript -e 'const Mail = Application("Mail"); JSON.stringify(Mail.mailboxes.byName("amail-ci").messages.subject())'

      - name: Build and run amail against it
        continue-on-error: true
        timeout-minutes: 3
        run: |
          go build -o amail .
          ./amail accounts

      - name: TCC decisions (questions 2 and 6)
        if: always()
        run: log show --last 15m --predicate 'subsystem == "com.apple.TCC"' --style compact | grep -i -E 'mail|AppleEvents' | tail -200 > tcc.log || true

      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: trial-${{ matrix.os }}
          path: |
            *.png
            tcc.log
```

Notes on the sketch: the row shape copies the image's own Safari rows (path client, `auth_value` 2, target bundle identifier, no code requirement), with columns named so a schema change fails loudly; the action versions and the exact JavaScript for creating a mailbox are unchecked and may need adjusting on the first run.
