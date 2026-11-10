<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/media/banner-dark.svg">
  <img src="docs/media/banner.svg" alt="mailslot: Apple Mail from the terminal, for the accounts other tools can't sign in to" width="100%">
</picture>

# mailslot

`mailslot` is a command line tool for macOS that reads, searches, triages and writes mail by scripting the Mail app, so it needs no password or token and works with every account Mail is already signed in to.

![Listing accounts, mailboxes and messages, then reading one](docs/media/read.gif)

_Every recording in this README runs against invented mail (the [fixture backend](CONTRIBUTING.md#fixture-backend)), not a real mailbox._

Arriving from `amail`? This is the same tool under its new name. v0.1.0 stays published as `amail`; everything after it is `mailslot`.

## Is this for you?

Yes, if:

- **You have a locked account.** Your university or company blocks third-party mail clients: no consent for outside OAuth apps, and no IMAP, basic authentication or app passwords. Apple Mail is usually still allowed, and `mailslot` only ever talks to Mail.
- **You want a script or an AI agent to work on your mail without holding credentials.** `mailslot` is Automation-only: the one macOS permission it needs is Automation control of Mail. It needs no Full Disk Access, no Accessibility and no secrets. Writing is draft-first, and two safety switches make `mailslot` refuse to change or send anything.

Use something else if:

- Your account allows IMAP, JMAP or an OAuth client of your choice. A server-direct client such as [himalaya](https://github.com/pimalaya/himalaya) or [aerc](https://aerc-mail.org) is faster and does more.
- You are not on macOS, or Mail is not set up on the Mac.
- You want a full-screen terminal mail client, HTML composing, or management of rules, VIPs and smart mailboxes.

## Install

`mailslot` needs macOS with the Mail app, and either an account in Mail or a mailbox under On My Mac.

| Way | State |
|---|---|
| `go install` | Works today from `main` (needs Go 1.25 or newer) |
| Homebrew | Planned in the `ChiragAgg5k/homebrew-tools` tap; not published yet |
| Release binaries | None under the name `mailslot` yet. The only release, v0.1.0, has the old `amail` binary |

```sh
go install github.com/ChiragAgg5k/mailslot@main
```

`@latest` starts working with the first `mailslot` release: until then it resolves to v0.1.0, which predates the rename and does not install. When binaries are published they will be unsigned until a signing certificate is set up, and the release notes will say so.

### First run

The first command makes macOS ask whether the app running it (your terminal) may control Mail. Allow it. If you dismissed the prompt, turn it on under System Settings → Privacy & Security → Automation.

The permission belongs to the app that hosts the command, not to `mailslot`. Some hosts, such as agent apps without the Apple Events entitlement, cannot be granted it at all; run `mailslot` from a terminal app instead. `mailslot doctor` checks all of this and changes nothing.

## Command tour

Every command takes `--account` (`-a`) and `--json`. The account comes from `--account`, then `MAILSLOT_ACCOUNT`, then the configuration file (`~/.config/mailslot/config.toml`, or under `XDG_CONFIG_HOME`). With a single enabled account there is nothing to pick. `mailslot <command> --help` lists every flag.

### Read

```sh
mailslot accounts                          # what Mail has, with local last
mailslot accounts use Work                 # make one the default
mailslot mailboxes                         # mailboxes with unread and total counts

mailslot list                              # newest 20 in the Inbox; * unread, ! flagged
mailslot list --unread -n 50
mailslot list -m 'Sent Items' -n 0         # every message

mailslot read 1006
mailslot read 1006 -m Archive
mailslot read 1006 --raw                   # the RFC 822 source
mailslot read 1006 --raw -o message.eml    # saved as a file

mailslot attachments 1006 -o ~/Downloads
```

### Search

![Searching, piping JSON through jq and reading by Message-ID](docs/media/search.gif)

The text matches the subject or the sender. Only the Inbox is searched unless you name other mailboxes; several are searched one after another and the results merged, newest first.

```sh
mailslot search invoice
mailslot search --from dana --since 2026-10-01 --unread
mailslot search --subject 'meeting' -m Inbox -m Archive -n 0
mailslot search planning --all-mailboxes
mailslot search INV-2041 --body --since 2026-10-01
```

`--body` also matches the text of each message. It reads every candidate message, which is slow, so it needs search text and `--since`.

With `--json` the output is made for other programs:

```sh
mailslot list --unread --json | jq -r '.[] | "\(.id)\t\(.messageId)\t\(.subject)"'
```

### Write

`send` and `reply` are draft-first. Without `--send` they open a draft window in Mail and nothing is sent to the recipients. Mail may still save the draft to the account's Drafts mailbox on the server, as it does with any draft. Before anything is sent, Mail's sender is read back and checked; it never falls back to another account.

```sh
mailslot send --to dana@example.com --subject 'Notes' --body 'See attached.' --attach notes.pdf
mailslot send --to dana@example.com --cc priya@example.com --subject 'Notes' --body-file draft.txt --send
git log -5 | mailslot send --to team@example.com --subject 'Latest commits' --body-file - --send
mailslot send --from sam@example.com --signature Work --to dana@example.com --subject 'Hello' --body 'Hi Dana'

mailslot reply 1006 --body 'Thanks, that works.'
mailslot reply 1006 --all --body-file answer.txt --send
```

- `--from` must be one of the account's addresses; the default is its first address.
- `--signature` appends one of Mail's signatures by name. There is no default signature.
- `--to`, `--cc`, `--bcc` and `--attach` repeat.
- A reply is a new message with the original quoted below your text. It threads by its `Re:` subject (see [Limits](#limits)).

### Triage

![A dry run, then archiving and marking, then the list showing the change](docs/media/triage.gif)

`mark`, `flag`, `move`, `archive` and `delete` change Mail, so they act only on the message references you list, never on "everything matching a filter". `--dry-run` prints what would change and changes nothing. Each prints one row per message with the change made and, after a move, the message's new id.

```sh
mailslot mark 1009 1010 --read
mailslot mark 1009 --unread
mailslot flag 1004 --color orange          # red, orange, yellow, green, blue, purple or gray
mailslot flag 1004 --clear
mailslot move 1004 --to Projects
mailslot archive 1008 1010 --dry-run
mailslot delete 1012 -m Archive
```

- `archive` moves to the account's archive mailbox (`Archive`, `Archives` or `All Mail`).
- `delete` moves to the account's Trash (`Trash`, `Deleted Items`, `Deleted Messages` or `Bin`). Nothing is ever erased: a message already in the Trash is left there with exit code 2, and emptying the Trash is up to you in Mail.
- `move` only moves between mailboxes of the same account.

### Automate

`wait` blocks until a new message from a sender or with a subject arrives, checking for new mail every `--interval` (default 10s) until `--timeout` (default 5m). Only messages that were not there on its first check count, and none received more than 2 minutes before `wait` started. `--code` prints just the one-time code found in the message.

```sh
mailslot wait --from no-reply@example.com --subject 'verification code' --code --timeout 2m
mailslot wait --subject 'Build finished' --interval 30s --json
```

`wait` exits 5 when the timeout passes and 4 when the message arrives without a code.

`merge` opens one draft per row of a CSV file. The first line names the columns, and `--to`, `--cc`, `--subject` and the body may use `{{column}}` placeholders. Every row is checked before any draft opens. Merge only makes drafts and never sends; `--dry-run` prints the rows it would make.

```console
$ cat people.csv
email,name
dana@example.com,Dana
priya@example.com,Priya
$ mailslot merge people.csv --to '{{email}}' --subject 'Hello {{name}}' --body-file note.txt --dry-run
Dry run; nothing changed.
ROW  TO                 SUBJECT
1    dana@example.com   Hello Dana
2    priya@example.com  Hello Priya
```

It also takes `--from`, `--signature`, `--attach` (the same file on every draft) and `--body`.

### Safety switches

![A draft without --send, both switches refusing, and doctor](docs/media/safety.gif)

Two switches limit what `mailslot` will do when a script or an agent runs it. Set them in the configuration file or the environment; either place turns them on.

```toml
# ~/.config/mailslot/config.toml
account = "Work"
read_only = true
send = false
```

- **Read-only** (`read_only = true` or `MAILSLOT_READ_ONLY=1`): `mark`, `flag`, `move`, `archive`, `delete`, `send`, `reply` and `merge` refuse with exit code 6. `--dry-run` still works. `wait` still runs: it asks Mail to check for new mail but changes no message.
- **No-send** (`send = false` or `MAILSLOT_NO_SEND=1`): `--send` refuses with exit code 6. Drafts still open for review.

The switches guard against mistakes; they are not a security boundary. They only apply to `mailslot` commands, read-only still lets mail be read, and a program that can unset an environment variable, edit the configuration file or run `osascript` itself can get around them. Limit what an untrusted agent can run by other means.

### Doctor

```sh
mailslot doctor
```

Checks that Mail is present and answering and that Automation is permitted, then lists the accounts, the configuration file and the state of both safety switches. It changes nothing.

## Message references

Every command that shows a message prints its numeric `id` and its `messageId` (the Message-ID header). Every command that takes a message accepts either.

- **Numeric id** (`1006`): fast, but it only holds inside the mailbox it was listed from, so pass the same `--mailbox`. Mail gives a message a new id when it moves.
- **Message-ID** (`standup-1009@mail.example.com`, with or without `<>`): durable. Without `--mailbox` it is looked up in every mailbox of the account, Inbox first.

A reference that no longer resolves exits with code 4. If it was a numeric id, retry with the Message-ID.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General failure |
| 2 | Usage: a bad flag, argument or combination |
| 3 | Automation permission: the app running the command is not allowed to control Mail |
| 4 | Not found: message, mailbox, account or signature |
| 5 | Busy: Mail did not answer in time, another `mailslot` held the lock, or `wait` timed out |
| 6 | Refused by a safety switch, or a sender check failed |

Errors go to standard error as `mailslot: <message>` and `hint: <hint>`. With `--json` they are one JSON object:

```json
{"error":"no message 9999 in mailbox \"Inbox\" of account \"Work\"","code":4,"hint":"a numeric id only holds inside the mailbox it was listed from and changes when the message moves; retry with its Message-ID (the messageId field)"}
```

Mail answers one request at a time, so concurrent `mailslot` commands queue on a lock file in `~/Library/Caches/mailslot`. A command that waits longer than `MAILSLOT_LOCK_TIMEOUT` (a Go duration, default `2m`) gives up with exit code 5.

## Local mailboxes

Mailboxes under On My Mac belong to the Mac, not to any account. Address them with the reserved pseudo-account `local`:

```sh
mailslot mailboxes --account local
mailslot list --account local -m Receipts
mailslot move 1022 --account local -m Receipts --to Archive
```

`local` works with zero accounts in Mail, and is picked automatically when there are none. It cannot send mail or wait for new mail.

## Agents

The JSON command line is the interface for agents; there is no MCP server mode. [`skills/mailslot/SKILL.md`](skills/mailslot/SKILL.md) is an agent skill that teaches safe use: draft-first, the safety switches, message references and exit codes. Copy the `skills/mailslot` directory to wherever your agent loads skills from.

## Other approaches

`mailslot` is deliberately narrow. This is how it sits next to the other ways of reaching mail from a terminal or an agent, as of November 2026.

| Approach | Examples | Sign-in and permissions | Works with a locked account | Good at |
|---|---|---|---|---|
| Server-direct clients | [himalaya](https://github.com/pimalaya/himalaya), [aerc](https://aerc-mail.org), neomutt | Their own sign-in: a password, or an OAuth client the organisation has approved | No, unless the organisation approves the client | Speed, full feature sets, any operating system |
| Apple Mail tools that read Mail's files | [mail.app-cli](https://github.com/0xSMW/mail.app-cli), [mailctl](https://github.com/zydtiger/mailctl), [amcli](https://github.com/joargp/amcli) | No credentials; Full Disk Access, some with Automation or Accessibility as well | Yes | Fast full-text search across every mailbox, wider coverage of Mail's features |
| Apple Mail MCP servers | [patrickfreyer/apple-mail-mcp](https://github.com/patrickfreyer/apple-mail-mcp), [sweetrb/apple-mail-mcp](https://github.com/sweetrb/apple-mail-mcp) | No credentials; Automation, some with optional IMAP passwords for speed | Yes, on their scripting path | Agents that speak MCP, many tools in one server |
| `mailslot` | | No credentials; Automation only | Yes | A single binary for shells, scripts and agents; draft-first writing; safety switches; fixed exit codes |

If one of the others fits your account and your permissions, it will probably do more than `mailslot` does.

## Limits

- **macOS only, tested on macOS 26 only.** Older versions may work; nothing is claimed about them.
- **Plain text only** when writing. Mail's scripting cannot set HTML content.
- **Replies are new messages.** Mail does not let scripts see the quoted text of its own reply window, so `mailslot reply` writes a fresh message with the original quoted below your text. It threads by subject in most clients but carries no `In-Reply-To` header.
- **Speed depends on the mailbox.** Every request goes through Mail's scripting interface, so listing and searching slow down as a mailbox grows, and `--body` search is slow enough to need `--since`. As a guide, on the author's Mac the newest 20 messages of a 13,000-message Gmail inbox list in about 2 seconds, and a subject or sender search of it takes about the same. Looking a message up by Message-ID in a mailbox that size takes several seconds.
- **Some write commands have not been checked against real accounts.** `send`, `reply`, `merge` and `wait` have only been run against the fixture backend and compiled against Mail's scripting dictionary. `mark`, `flag`, `move`, `archive` and `delete` run against a real Mail app in CI, but only on local mailboxes, not on a mail account. Use drafts and `--dry-run` first, and check the result in Mail.
- **Flag colour order is unconfirmed.** Mail's scripting dictionary only documents a flag index; the mapping from colour names follows Mail's Flag menu and is still being checked.
- **Exchange quirks.** `accounts` shows the kind as `unknown`, because Mail's scripting dictionary has no name for Exchange. `read --raw` may be incomplete when Mail has not downloaded the whole message.
- **Numeric ids belong to this Mac** and change when a message moves. The Message-ID travels; the numeric id does not.
- **Search reads bodies only with `--body`.** By default it matches subject and sender.

## How it works

Each command runs a small JavaScript for Automation script through `osascript`, which asks the Mail app and prints JSON for the Go program to format. The scripts ask Mail for whole columns at once (every subject of the matching messages in one request) instead of walking message by message. They live in [`internal/mail/script`](internal/mail/script).

`mailslot` never reads Mail's files on disk and never talks to a mail server. Whatever Mail can do with an account, it asks Mail to do.

## Testing

```sh
go test ./...
```

The command tests run against the fixture backend and never touch Mail, and every embedded script is compiled with `osacompile`. An opt-in end-to-end suite drives the real Mail app; CI runs it against local mailboxes on hosted macOS runners. [CONTRIBUTING.md](CONTRIBUTING.md) explains the three layers, the fixture backend and how the recordings are made.

## Licence

[MIT](LICENSE)
