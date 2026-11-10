# mailslot

Apple Mail from the terminal. `mailslot` reads, searches, triages and writes mail by scripting the Mail app that ships with macOS, so it works with every account Mail can sign in to: iCloud, Gmail, Exchange, Microsoft 365 and plain IMAP.

```console
$ mailslot list -n 3
ID       RECEIVED          FROM            SUBJECT
4812  *  2026-11-10 09:01  Ada Lovelace    Notes on the analytical engine
4810     2026-11-09 17:40  GitHub          [mailslot] Run failed: CI
4807  !  2026-11-09 11:15  Grace Hopper    Re: compiler meeting
```

## Why

Terminal mail clients talk to the mail server themselves, which means each one needs its own sign-in. Many universities and companies run a locked account: they only allow sign-in from an approved list of mail apps, and third-party clients are not on it. Apple Mail usually is. `mailslot` never touches a password, a token or a server: it asks the Mail app, which is already signed in. It is Automation-only: the one macOS permission it needs is Automation control of Mail.

It is also built to be handed to scripts and AI agents:

- every command prints JSON with `--json`, errors included;
- messages are named by a message reference (Mail's numeric id or the Message-ID header), not by a position that shifts when new mail arrives;
- writing is draft-first: a message opens as a draft for you to review, and only sends when you pass `--send`;
- changes act only on the messages you name, `--dry-run` previews them, and safety switches can turn writing or sending off entirely;
- exit codes say what went wrong, so a caller can tell "not found" from "Mail is busy" from "not allowed".

## Install

Requires macOS. Mail needs at least one account, or at least one mailbox under On My Mac.

```sh
go install github.com/ChiragAgg5k/mailslot@latest
```

Or download a binary for your Mac from the [releases page](https://github.com/ChiragAgg5k/mailslot/releases), unpack it and move `mailslot` somewhere on your `PATH`.

The first command makes macOS ask whether your terminal may control Mail. Allow it. If you dismissed the prompt, turn it on under System Settings → Privacy & Security → Automation. Some apps, such as agent hosts without the Apple Events entitlement, cannot be granted this permission at all; run `mailslot` from a terminal app instead. `mailslot doctor` checks all of this.

## Use

Every command takes `--account` (`-a`) and `--json`. The account comes from `--account`, then `MAILSLOT_ACCOUNT`, then the configuration file (`~/.config/mailslot/config.toml`, or under `XDG_CONFIG_HOME`). With a single enabled account there is nothing to pick.

### accounts

```sh
mailslot accounts                 # what Mail has, with local last
mailslot accounts use iCloud      # make one the default
```

### mailboxes

```sh
mailslot mailboxes                # mailboxes, with unread and total counts
mailslot mailboxes --account local
```

### list

The newest messages of one mailbox. `*` marks unread, `!` flagged.

```sh
mailslot list                              # newest 20 in the Inbox
mailslot list --unread -n 50
mailslot list -m 'Sent Items' -n 0         # every message
```

### read

```sh
mailslot read 4812
mailslot read 4812 -m Archive
mailslot read 4812 --raw                   # the RFC 822 source
mailslot read 4812 --raw -o message.eml    # saved as a file
```

### attachments

```sh
mailslot attachments 4812 -o ~/Downloads
```

### search

The text matches the subject or the sender. Only the Inbox is searched unless you name other mailboxes; several are searched one after another and the results merged, newest first.

```sh
mailslot search invoice
mailslot search --from github --since 2026-11-01 --unread
mailslot search --subject 'meeting' -m Inbox -m Archive -n 0
mailslot search planning --all-mailboxes
mailslot search INV-2041 --body --since 2026-10-01
```

`--body` also matches the text of each message. It reads every candidate message, which is slow, so it needs search text and `--since`.

### send and reply

Both are draft-first. Without `--send` they open a draft window in Mail and nothing leaves the Mac. Mail's sender is read back and checked before anything is sent; it never falls back to another account.

```sh
mailslot send --to ada@example.com --subject 'Notes' --body 'See attached.' --attach notes.pdf
mailslot send --to ada@example.com --cc grace@example.com --subject 'Notes' --body-file draft.txt --send
git log -5 | mailslot send --to team@example.com --subject 'Latest commits' --body-file - --send
mailslot send --from sam@example.com --signature Work --to ada@example.com --subject 'Hello' --body 'Hi Ada'

mailslot reply 4812 --body 'Thanks, that works.'
mailslot reply 4812 --all --body-file answer.txt --send
```

- `--from` must be one of the account's addresses; the default is its first address.
- `--signature` appends one of Mail's signatures by name. There is no default signature.
- `--to`, `--cc`, `--bcc` and `--attach` repeat.
- A reply is a new message with the original quoted below your text. It threads by its `Re:` subject (see [Limits](#limits)).

### mark, flag, move, archive, delete

These change Mail, so they act only on the message references you list, never on "everything matching a filter". `--dry-run` prints what would change and changes nothing. Each prints one row per message with the change made and, after a move, the message's new id.

```sh
mailslot mark 4812 4810 --read
mailslot mark 4812 --unread
mailslot flag 4812 --color orange          # red, orange, yellow, green, blue, purple or gray
mailslot flag 4812 --clear
mailslot move 4812 --to Projects
mailslot archive 4812 4810 --dry-run
mailslot delete 4807 -m Archive
```

- `archive` moves to the account's archive mailbox (`Archive`, `Archives` or `All Mail`).
- `delete` moves to the account's Trash (`Trash`, `Deleted Items`, `Deleted Messages` or `Bin`). Nothing is ever erased: a message already in the Trash is left there, and emptying the Trash is up to you in Mail.
- `move` only moves between mailboxes of the same account.

### wait

Blocks until a new message from a sender or with a subject arrives, checking for new mail every `--interval` (default 10s) until `--timeout` (default 5m). Only messages received after `wait` started count. `--code` prints just the one-time code found in the message.

```sh
mailslot wait --from no-reply@example.com --subject 'verification code' --code --timeout 2m
mailslot wait --subject 'Build finished' --interval 30s --json
```

`wait` exits 5 when the timeout passes and 4 when the message arrives without a code.

### merge

Opens one draft per row of a CSV file. The first line names the columns, and `--to`, `--cc`, `--subject` and the body may use `{{column}}` placeholders. Every row is checked before any draft opens. Merge only makes drafts and never sends; `--dry-run` prints the rows it would make.

```console
$ cat people.csv
email,name
ada@example.com,Ada
grace@example.com,Grace
$ mailslot merge people.csv --to '{{email}}' --subject 'Hello {{name}}' --body-file note.txt --dry-run
Dry run; nothing changed.
ROW  TO                 SUBJECT
1    ada@example.com    Hello Ada
2    grace@example.com  Hello Grace
```

It also takes `--from`, `--signature`, `--attach` (same file on every draft) and `--body`.

### doctor

Checks that Mail is present and answering, that Automation is permitted, and lists the accounts, the configuration file and the safety switches. It changes nothing.

```sh
mailslot doctor
```

### Script it

```sh
mailslot list --unread --json | jq -r '.[] | "\(.id)\t\(.messageId)\t\(.subject)"'
```

## Message references

Every command that shows a message prints its numeric `id` and its `messageId` (the Message-ID header). Every command that takes a message accepts either.

- **Numeric id** (`4812`): fast, but it only holds inside the mailbox it was listed from, so pass the same `--mailbox`. Mail gives a message a new id when it moves.
- **Message-ID** (`quarterly-1@mail.example.com`, with or without `<>`): durable. Without `--mailbox` it is looked up in every mailbox of the account, Inbox first.

A reference that no longer resolves exits with code 4. If it was a numeric id, retry with the Message-ID.

## Errors and exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General failure |
| 2 | Usage: a bad flag, argument or combination |
| 3 | Automation permission: this app is not allowed to control Mail |
| 4 | Not found: message, mailbox, account or signature |
| 5 | Busy: Mail did not answer in time, another `mailslot` held the lock, or `wait` timed out |
| 6 | Refused by a safety switch, or a sender check failed |

Errors go to standard error as `mailslot: <message>` and `hint: <hint>`. With `--json` they are one JSON object:

```json
{"error":"no message 9999 in mailbox \"Inbox\" of account \"Work\"","code":4,"hint":"a numeric id only holds inside the mailbox it was listed from and changes when the message moves; retry with its Message-ID (the messageId field)"}
```

## Safety switches

Two switches make `mailslot` safe to hand to an agent. Set them in the configuration file or the environment; either place turns them on.

```toml
# ~/.config/mailslot/config.toml
account = "iCloud"
read_only = true
send = false
```

- **Read-only** (`read_only = true` or `MAILSLOT_READ_ONLY=1`): `mark`, `flag`, `move`, `archive`, `delete`, `send`, `reply` and `merge` refuse with exit code 6. `--dry-run` still works.
- **No send** (`send = false` or `MAILSLOT_NO_SEND=1`): `--send` refuses with exit code 6. Drafts still open for review.

## Local mailboxes

Mailboxes under On My Mac belong to the Mac, not to any account. Address them with the reserved pseudo-account `local`:

```sh
mailslot list --account local -m Receipts
mailslot move 2001 --account local -m Receipts --to Archive
```

`local` works with zero accounts in Mail, and is picked automatically when there are none. It cannot send mail or wait for new mail.

## One request at a time

Mail answers one request at a time, so concurrent `mailslot` commands queue on a lock file in `~/Library/Caches/mailslot`. A command that waits longer than `MAILSLOT_LOCK_TIMEOUT` (a Go duration, default `2m`) gives up with exit code 5.

## Fixture backend

Setting `MAILSLOT_FIXTURE` to a directory of JSON files makes `mailslot` use that data instead of Mail. Nothing reaches Mail, and it runs off macOS too, which makes it the safe way to demo, script against or test the tool.

```sh
MAILSLOT_FIXTURE=testdata/fixture mailslot --account Work list
MAILSLOT_FIXTURE=testdata/local mailslot mailboxes            # zero accounts, local mailboxes only
```

- `testdata/fixture` holds invented accounts (`Work`, `Personal`, `Old`), their mailboxes, local mailboxes and messages; `incoming.json` holds mail that arrives when `wait` checks for new mail.
- Changes are kept in memory and lost when the command ends. Set `MAILSLOT_FIXTURE_STATE` to a file outside the fixture directory to carry them from one command to the next; the fixture directory itself is never written. Use one state file per fixture directory.
- `MAILSLOT_FIXTURE_FAULT=permission` or `timeout` makes every call fail as a denied Automation permission (exit 3) or an unanswered Mail (exit 5).

## Testing

```sh
go test ./...
```

The unit and command tests run against the fixture backend and never touch Mail.

The end-to-end suite drives the real Mail app and changes the mailboxes it is given. It is opt-in: run it only on a dedicated account or local mailboxes that hold nothing you care about.

```sh
MAILSLOT_E2E_ACCOUNT=local go test -tags e2e ./internal/e2e/ -v -count=1
```

CI runs it in the `e2e-local` job on hosted macOS runners, which have no accounts: it imports the fixture into local mailboxes and tests against them.

## Limits

- **macOS only**, and Mail has to be able to run. `mailslot` starts it when needed.
- **Search reads bodies only with `--body`**, which is slow and needs `--since`. By default it matches subject and sender.
- **Replies are new messages.** Mail does not let scripts see the quoted text of its own reply window, so `mailslot reply` writes a fresh message with the original quoted below your text. It threads by subject in most clients but carries no `In-Reply-To` header.
- **`read --raw` may be incomplete on Exchange** when Mail has not downloaded the whole message.
- **Flag colour order is unconfirmed.** Mail's scripting dictionary only documents the flag index; the mapping from colour names follows Mail's Flag menu and is still being checked against Mail.
- **Message IDs belong to this Mac.** They are Mail's local identifiers, not something another machine understands, and they change when a message moves. The Message-ID travels; the numeric id does not.
- **Plain text only** when writing.

## How it works

Each command runs a small JavaScript for Automation script through `osascript`. The scripts ask Mail for whole columns at once (every subject of the matching messages in one request) instead of walking message by message, which is what keeps it usable on real mailboxes. The scripts live in [`internal/mail/script`](internal/mail/script).

## Licence

[MIT](LICENSE)
