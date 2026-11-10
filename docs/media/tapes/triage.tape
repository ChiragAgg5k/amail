# Triage: a dry run, then the change, then the list showing it.
# Recorded against the fixture backend, never against the Mail app.
# Run through docs/media/record.sh, which builds mailslot and prepares /tmp/mailslot-demo.
# MAILSLOT_FIXTURE_STATE (set in env.sh) carries the changes from one command to the next.
Output docs/media/triage.gif

Set Shell "bash"
Set FontSize 15
Set LetterSpacing 0
Set Width 1060
Set Height 640
Set Padding 24
Set Theme "GitHub Dark"
Set TypingSpeed 28ms

Hide
Type "source /tmp/mailslot-demo/env.sh || exit 1" Enter
Type "clear" Enter
Sleep 500ms
Show

Type "mailslot list -n 5"
Sleep 300ms
Enter
Sleep 2s

Type "mailslot archive 1008 1010 --dry-run"
Sleep 300ms
Enter
Sleep 2.5s

Type "mailslot archive 1008 1010"
Sleep 300ms
Enter
Sleep 2s

Type "mailslot mark 1009 --read"
Sleep 300ms
Enter
Sleep 1.5s

Type "mailslot list -n 5"
Sleep 300ms
Enter
Sleep 4s
