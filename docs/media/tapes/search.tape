# Search, JSON output through jq, and reading by Message-ID.
# Recorded against the fixture backend, never against the Mail app.
# Run through docs/media/record.sh, which builds mailslot and prepares /tmp/mailslot-demo.
Output docs/media/search.gif

Require jq

Set Shell "bash"
Set FontSize 15
Set LetterSpacing 0
Set Width 1060
Set Height 660
Set Padding 24
Set Theme "GitHub Dark"
Set TypingSpeed 28ms

Hide
Type "source /tmp/mailslot-demo/env.sh || exit 1" Enter
Type "clear" Enter
Sleep 500ms
Show

Type "mailslot search planning --all-mailboxes"
Sleep 300ms
Enter
Sleep 2.2s

Type "mailslot search --from dana --since 2026-10-06 --json | jq '.[0]'"
Sleep 300ms
Enter
Sleep 3s

Type `mailslot search --unread --json | jq -r '.[] | "\(.id)  \(.messageId)"'`
Sleep 300ms
Enter
Sleep 2.5s

Type "mailslot read standup-1009@mail.example.com"
Sleep 300ms
Enter
Sleep 4s
