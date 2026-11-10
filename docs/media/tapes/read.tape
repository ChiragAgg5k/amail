# Accounts, mailboxes, list and read.
# Recorded against the fixture backend, never against the Mail app.
# Run through docs/media/record.sh, which builds mailslot and prepares /tmp/mailslot-demo.
Output docs/media/read.gif

Set Shell "bash"
Set FontSize 15
Set LetterSpacing 0
Set Width 1060
Set Height 600
Set Padding 24
Set Theme "GitHub Dark"
Set TypingSpeed 28ms

Hide
Type "source /tmp/mailslot-demo/env.sh || exit 1" Enter
Type "clear" Enter
Sleep 500ms
Show

Type "mailslot accounts"
Sleep 300ms
Enter
Sleep 1.5s

Type "mailslot accounts use Work"
Sleep 300ms
Enter
Sleep 1.2s

Type "mailslot mailboxes"
Sleep 300ms
Enter
Sleep 2.2s

Type "clear"
Sleep 300ms
Enter
Sleep 400ms

Type "mailslot list -n 5"
Sleep 300ms
Enter
Sleep 2.2s

Type "mailslot read 1006"
Sleep 300ms
Enter
Sleep 4s
