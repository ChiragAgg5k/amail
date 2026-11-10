# amail

A command line tool that reads and writes mail by asking the Apple Mail app, for accounts that other terminal mail tools cannot sign in to.

## Language

**Locked account**:
A mail account whose organisation blocks third-party sign-in: no consent for outside OAuth apps, and no IMAP, basic authentication or app passwords.
_Avoid_: Restricted tenant, managed account

**Automation-only**:
Needing no macOS permission beyond Automation control of Mail: no Full Disk Access, no Accessibility, no credentials.
_Avoid_: Scripting-only, permissionless

**Draft-first**:
Writing a message leaves it as a draft for a person to review; sending is a separate, explicit choice.
_Avoid_: Safe mode, dry run

**Message reference**:
What a caller passes to name one message.
_Avoid_: Index, position, row
