# Safety: a draft without --send, both safety switches refusing, and doctor.
# Recorded against the fixture backend, never against the Mail app.
# Run through docs/media/record.sh, which builds mailslot and prepares /tmp/mailslot-demo.
Output docs/media/safety.gif

Set Shell "bash"
Set FontSize 15
Set LetterSpacing 0
Set Width 1060
Set Height 600
Set Padding 24
Set Theme "GitHub Dark"
Set TypingSpeed 20ms

Hide
Type "source /tmp/mailslot-demo/env.sh || exit 1" Enter
Type "clear" Enter
Sleep 500ms
Show

Type "mailslot send --to dana.okafor@example.com --subject 'Budget' --body 'Numbers follow tomorrow.'"
Sleep 300ms
Enter
Sleep 1.8s

Type "mailslot list -m Drafts -n 2"
Sleep 300ms
Enter
Sleep 1.6s

Type "MAILSLOT_NO_SEND=1 mailslot reply 1006 --body 'Thanks, Priya.' --send; echo exit $?"
Sleep 300ms
Enter
Sleep 2s

Type "MAILSLOT_READ_ONLY=1 mailslot delete 1004; echo exit $?"
Sleep 300ms
Enter
Sleep 2s

Type "MAILSLOT_READ_ONLY=1 mailslot doctor"
Sleep 300ms
Enter
Sleep 3.5s
