# amail

Apple Mail from the terminal. `amail` reads, searches and writes mail by scripting the Mail app that ships with macOS, so it works with every account Mail can sign in to: iCloud, Gmail, Exchange, Microsoft 365 and plain IMAP.

```console
$ amail list -n 3
ID       RECEIVED          FROM            SUBJECT
4812  *  2026-11-10 09:01  Ada Lovelace    Notes on the analytical engine
4810     2026-11-09 17:40  GitHub          [amail] Run failed: CI
4807  !  2026-11-09 11:15  Grace Hopper    Re: compiler meeting
```

## Why

Terminal mail clients talk to the mail server themselves, which means each one needs its own sign-in. Many universities and companies only allow sign-in from an approved list of mail apps, and third-party clients are not on it. Apple Mail usually is. `amail` never touches a password, a token or a server: it asks the Mail app, which is already signed in.

It is also built to be handed to scripts and AI agents:

- every command prints JSON with `--json`;
- messages are addressed by Mail's own stable ID, not by a position that shifts when new mail arrives;
- writing a message opens a draft for you to review, and only sends when you pass `--send`.

## Install

Requires macOS with at least one account set up in Mail.

```sh
go install github.com/ChiragAgg5k/amail@latest
```

Or download a binary for your Mac from the [releases page](https://github.com/ChiragAgg5k/amail/releases), unpack it and move `amail` somewhere on your `PATH`.

The first command makes macOS ask whether your terminal may control Mail. Allow it. If you dismissed the prompt, turn it on under System Settings → Privacy & Security → Automation.

## Use

Pick the account once, or pass `--account` (or set `AMAIL_ACCOUNT`) on any command. With a single account there is nothing to pick.

```sh
amail accounts                 # what Mail has
amail accounts use iCloud      # make one the default
amail mailboxes                # folders, with unread and total counts
```

Read:

```sh
amail list                              # newest 20 in the inbox
amail list --unread -n 50
amail list -m 'Sent Items'
amail read 4812
amail attachments 4812 -o ~/Downloads
```

Search one mailbox by subject or sender, optionally narrowed:

```sh
amail search invoice
amail search --from github --since 2026-11-01 --unread
amail search 'meeting' -m Archive -n 0   # every match
```

Write. Without `--send` these open a draft window in Mail and send nothing:

```sh
amail send --to ada@example.com --subject 'Notes' --body 'See attached.' --attach notes.pdf
amail send --to ada@example.com --cc grace@example.com --subject 'Notes' --body-file draft.txt --send
git log -5 | amail send --to team@example.com --subject 'Latest commits' --body-file - --send

amail reply 4812 --body 'Thanks, that works.'
amail reply 4812 --all --body-file answer.txt --send
```

Script it:

```sh
amail list --unread --json | jq -r '.[] | "\(.id)\t\(.subject)"'
```

## Limits

- **macOS only**, and Mail has to be able to run. `amail` starts it when needed.
- **Search does not look inside message bodies**, only at subject and sender, and only in one mailbox at a time.
- **Replies are new messages.** Mail does not let scripts see the quoted text of its own reply window, so `amail reply` writes a fresh message with the original quoted below your text. It threads by subject in most clients but carries no `In-Reply-To` header.
- **Large mailboxes are slower.** Each request is a scan inside Mail; a listing takes a few seconds on a mailbox with tens of thousands of messages.
- **Message IDs belong to this Mac.** They stay the same between runs, but they are Mail's local identifiers, not something another machine understands.
- **Plain text only** when writing.

## How it works

Each command runs a small JavaScript for Automation script through `osascript`. The scripts ask Mail for whole columns at once (every subject of the matching messages in one request) instead of walking message by message, which is what keeps it usable on real mailboxes. The scripts live in [`internal/mail/script`](internal/mail/script).

## Licence

[MIT](LICENSE)
