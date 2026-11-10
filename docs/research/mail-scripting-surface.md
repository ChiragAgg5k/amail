# Mail scripting surface survey

Gathered on 2026-11-10 while charting the v1.0 map, on macOS 26.6.2 with Mail 16.0. Probes were read-only.

## Environment

- macOS 26.6.2 (build 25G83), Mail 16.0 (3864.700.51.1.1); Mail was running and no Automation prompt blocked `osascript`.
- `ls ~/Library/Mail` fails with `Operation not permitted`, as do `~/Library/Mail/V10` and `~/Library/Containers/com.apple.mail`. The `V*` glob matched nothing.
- The version folder is `V10`, learned from the scriptable `accountDirectory` property rather than the filesystem.

## 1. Dictionary reference

Properties are read-write unless marked (r).

**Commands**

| Command | Parameters | Returns |
|---|---|---|
| `delete` | specifier | - |
| `duplicate` | specifier, `to` location (optional), `with properties` (optional) | - |
| `move` | specifier, `to` location | - |
| `check for new mail` | `for` account (optional) | - |
| `synchronize` | `with` account | - |
| `reply` | message, `opening window` bool (optional), `reply to all` bool (optional) | outgoing message |
| `forward`, `redirect` | message, `opening window` bool (optional) | outgoing message |
| `send` | outgoing message | boolean |
| `extract name from`, `extract address from` | text | text |
| `mailto`, `GetURL` | text (mailto URL only) | - |
| `import Mail mailbox` | `at` file | - |
| `perform mail action with messages` | list of message, `in mailboxes` (optional), `for rule` (optional) | - (a handler Mail calls in rule scripts) |
| `bounce` | message | hidden, "does nothing" |

Classes also respond to: `save`, `close`, `send` (outgoing message); `open` (message); `save` (mail attachment).

**Classes**

- **application**
  - Elements: account, pop/imap/iCloud account, smtp server, outgoing message, mailbox (the "On My Mac" ones), message viewer, rule, signature.
  - Read-only: `selection`, `inbox`, `drafts mailbox`, `sent mailbox`, `trash mailbox`, `junk mailbox`, `outbox`, `primary email`, `application version`, `background activity count`.
  - Read-write preferences: `always bcc/cc myself`, `fetch interval`, `fetches automatically`, `choose signature when composing`, `selected signature`, `default message format`, `quote original message`, `include all original message text`, `same reply format`, `download html attachments`, `expand group addresses`, `highlight selected conversation`, `new mail sound`, plus fonts and quoting colours.
- **message**
  - Elements: to/cc/bcc recipient, recipient, header, mail attachment.
  - Read-only: `id` (integer), `all headers`, `content` (rich text), `date received`, `date sent`, `message id`, `source`, `reply to`, `message size`, `sender`, `subject`, `was forwarded`, `was redirected`, `was replied to`.
  - Read-write: `mailbox`, `read status`, `flagged status`, `flag index` (-1 means none), `junk mail status`, `deleted status`, `background color`.
- **outgoing message**
  - Elements: to/cc/bcc recipient, recipient.
  - `sender`, `subject`, `visible`, `message signature`, `id` (r), plus `content` (settable in practice; amail does so) and `attachment` elements via the Text Suite.
  - Hidden, write-only, "does nothing (deprecated)": `html content`, `vcard path`.
- **mailbox** (and `container`): `name`, `unread count` (r), `account` (r), `container` (r); elements mailbox, message.
- **account**
  - `name`, `id` (r), `account type` (r), `email addresses`, `full name`, `user name`, `server name`, `port`, `uses ssl`, `authentication`, `enabled`, `delivery account`, `account directory` (r), `password` (write-only in effect).
  - `move deleted messages to trash`, and the empty-trash / empty-junk frequency and on-quit settings.
  - **imap account** adds `message caching`, `compact mailboxes when closing`, and four `store … on server` flags. **iCloud account** adds nothing. **pop account** adds four deletion and size settings.
- **smtp server**: `name` (r), `account type` (r), `server name`, `port`, `user name`, `password`, `authentication`, `enabled`, `uses ssl`.
- **message viewer**
  - Element: message.
  - Read-only: `id` and the six special mailboxes.
  - Read-write: `selected messages`, `selected mailboxes`, `visible messages`, `sort column`, `sorted ascending`, `visible columns`, `mailbox list visible`, `preview pane is visible`.
- **rule**
  - Element: rule condition.
  - `name`, `enabled`, `all conditions must be met`, `stop evaluating rules`.
  - Actions: `mark read`, `mark flagged`, `mark flag index`, `delete message`, `color message`, `highlight text using color`, `move message` + `should move message`, `copy message` + `should copy message`, `forward message`, `forward text`, `redirect message`, `reply text`, `play sound`, `run script`.
- **rule condition**: `rule type`, `qualifier`, `header`, `expression`.
- **signature**: `name`, `content`.
- **recipient** (to/cc/bcc): `name`, `address`.
- **header**: `name`, `content`.
- **mail attachment** (all read-only): `name`, `MIME type`, `file size`, `downloaded`, `id`.
- **Text Suite**: rich text, paragraph, word, character, attribute run (`color`, `font`, `size`); attachment (`file name`).
- **Deprecated**: `ldap server`, `OLD message editor`.

**Enumerations**

- `TypeOfAccount`: pop, smtp, imap, iCloud, unknown.
- `Authentication`: password, apop, kerberos 5, ntlm, md5, external, Apple token, none.
- `HighlightColors`: blue, gray, green, none, orange, other, purple, red, yellow.
- `QuotingColor`: the same colours without gray and none.
- `DefaultMessageFormat`: plain format, rich format.
- `MessageCachingPolicy`: four values.
- `RuleQualifier`: begins with, does contain, does not contain, ends with, equal to, less than, greater than, none.
- `RuleType`: account, any recipient, cc header, matches every message, from header, header key, message content, message is junk mail, sender is / is not in my contacts, in my previous recipients, member of group, sender is VIP, subject header, to header, to or cc header, attachment type.
- `ViewerColumns`: 13 columns.
- `HeaderDetail`, `LdapScope`: deprecated.

## 2. Capability table

"Uses" means the amail scripts in `internal/mail/script/` call it today.

| Feature | In dictionary | Property / command | amail uses |
|---|---|---|---|
| Mark read/unread | Yes | `read status` (rw) | Reads only |
| Flag, flag colour | Yes | `flagged status`, `flag index` (colour-to-index mapping not documented) | Reads `flaggedStatus` only |
| Junk status | Yes | `junk mail status` (rw) | No |
| Move | Yes | `move … to mailbox`, or set `mailbox` | No |
| Copy | Yes | `duplicate … to mailbox` | No |
| Archive | No dedicated command | Move to the account's Archive mailbox, found by name | No |
| Delete | Yes | `delete`, or `deleted status` | No |
| Create / rename / delete mailbox | Yes (generic) | `make new mailbox`, `name` (rw), `delete` | No |
| Forward, redirect | Yes | `forward`, `redirect` return an outgoing message | No |
| Reply with native threading | Yes | `reply` (`reply to all`, `opening window`); the result exposes only sender, subject, recipients, signature, visible, id | No; `draft.go` builds a new "Re:" message with no In-Reply-To |
| HTML when composing | No | `html content` is hidden and "does nothing" | No |
| HTML body / raw source | Yes | `source` (r) | No |
| All headers | Yes | `all headers`, `header` elements | No |
| Message-ID | Yes | `message id` | Yes (`read.js`) |
| In-Reply-To / References | Via headers only | `headers.byName(…)` or parse `all headers` | No |
| Conversation grouping | No | Rebuild from References / In-Reply-To; `highlight selected conversation` is only a preference | No |
| Signatures: list, choose | Yes | `signatures`, `message signature`, `selected signature` | No |
| Rules: list, toggle, create | Yes | `rule`, `rule condition`, `enabled`, `make new rule` | No |
| Run a rule on demand | No | `perform mail action with messages` is a handler Mail invokes, not a way to apply rules | No |
| VIPs | No | Only `sender is VIP` as a rule type | No |
| Smart mailboxes | No | No class | No |
| Check for new mail | Yes | `check for new mail [for account]`, `synchronize with account` | No |
| Account properties | Yes | `email addresses`, `server name`, `enabled`, `account type`, and more | Uses name, type, enabled, addresses |
| Message size | Yes | `message size` | No |
| Attachment properties | Yes | `name`, `MIME type`, `file size`, `downloaded`, `id`, `save` | Uses name, size, MIME type; not `downloaded` |
| Send from a chosen address | Yes | `sender` on outgoing message | Partly: always the account's first address |
| Import mailbox | Yes | `import Mail mailbox at file` | No |
| Extract name / address | Yes | `extract name from`, `extract address from` | No |
| `mailto` / `GetURL` | Yes | mailto URLs only; opens a compose window | No |
| `message://` URLs | Not in dictionary | System URL scheme, via `open "message://%3C<id>%3E"` | No |
| Selection | Yes | `selection` (r) | No |
| Message viewers | Yes | `selected messages`, `selected mailboxes`, `messages` | No |

## 3. Probe results

All probes were read-only and printed shapes only.

| Probe | Result | Time |
|---|---|---|
| `source()` | A string starting with a header line, with MIME-Version, Content-Type, a text/html part and transfer encodings; LF line endings. Google: 11,817 chars against `messageSize` 12,386. Exchange: 19,067 chars against 88,531, with no multipart boundary. | Exchange 14 ms, Google 589 ms |
| `allHeaders()` | Works. 7,568 chars / 46 distinct keys on Exchange, 5,369 / 25 on Google; includes Message-ID and Received. | 15 ms / 65 ms |
| `headers` elements | Work: 51 and 27 headers. `headers.byName('message-id').content()` and `whose({name:…})` both work. On an Exchange reply, In-Reply-To and References read as strings and `wasRepliedTo` was true. | 8–64 ms |
| `messageId()` | Works. Returned without angle brackets; the header element returns it with them (two characters longer). | 8 ms / 65 ms |
| `messages.whose({messageId})` | Found 1 match. | 87 ms (512 messages), 1,715 ms (13,372) |
| Bulk `messageId()` over the inbox | Works. | 204 ms / 3,440 ms |
| Bulk `messageSize()`, `flagIndex()` | Work. | 5–141 ms |
| Flag properties | `flagIndex` -1, plus junk, deleted, `backgroundColor` and the three `was…` flags all read. | 75 ms / 568 ms |
| `content()` | Returns a string. | about 300 ms |
| `selection()` | Returned an array of 1, because a message was selected in Mail. The nothing-selected case is untested; I did not change the selection. | 8 ms |
| `messageViewers` | 1 viewer. Readable: `id`, `sortedAscending`, `mailboxListVisible`, `previewPaneIsVisible`, `selectedMessages`, `selectedMailboxes`, `messages.length` (13,882). Failing with "AppleEvent handler failed": `sortColumn`, `visibleColumns`, `visibleMessages`. | 84 ms |
| Rules | 1 rule with 19 conditions. All probed properties and the first condition's four fields read; bulk `rules.enabled()` works. | 151 ms |
| Signatures | 0 defined. `Mail.signatures()` returns `null`, not `[]`, so `.length` on the result throws; `Mail.signatures.length` gives 0. Property reads untested. `selectedSignature()` returns a string. | 8–73 ms |
| Accounts | Google is `imap`; Exchange is `unknown`. `serverName()` is null for Exchange and `deliveryAccount()` fails. One smtp server. | 109 ms |
| Attachments (Exchange) | `name`, `fileSize`, `downloaded`, `id` read. `mimeType()` fails with "AppleEvent handler failed" on all three attachments tried. | 123 ms |
| Attachment scan (Google) | Counting `mailAttachments` on 40 messages found none, so MIME type on Google is untested. | 5,305 ms |
| App preferences | `fetchInterval()` returns -1. `fetchesAutomatically()`, `backgroundActivityCount()` and `primaryEmail()` fail. | 7–18 ms |
| `extractNameFrom`, `extractAddressFrom` | Work on a dummy address. | 8 ms each |
| Unified `Mail.inbox` | 13,884 messages, child mailboxes for both accounts. | 68 ms |

Not probed because they mutate or open UI: `reply`, `forward`, `move`, `make`, `check for new mail`, and opening `message://` URLs.

## 4. Documented but broken or deprecated

- **Declared dead in the dictionary itself**: `html content`, `vcard path`, `bounce`, `ldap server`, `OLD message editor`, `header detail`, `use address completion`, `use keychain`, and several account "empty sent messages" properties.
- **`reply` content**: reading the content of a reply returns an empty string, and setting it adds to the quoted original rather than replacing it. The reports are old (2011–2013, one workaround from Ventura 13.3); I found nothing confirming a fix. Workarounds mentioned are `without opening window`, or injecting text through a temporary signature. Sources: [applescript-users 2011](https://lists.apple.com/archives/applescript-users/2011/Nov/msg00086.html), [Late Night Software forum](https://forum.latenightsw.com/t/replying-to-email-messages/1969), [MacScripter](https://www.macscripter.net/t/add-content-to-reply-message-in-apple-mail/70134).
- **HTML compose**: reports say `html content` sends blank mail or works only with `visible` false, and that it has been broken since around El Capitan. Sources: [Moth Software](https://www.mothsoftware.com/almost-making-a-html-email-in-applescript-for-mail), [Automators](https://talk.automators.fm/t/applescript-to-send-html-email-via-mail-app/7904), [Apple developer forums](https://developer.apple.com/forums/thread/738842).
- **Message viewer**: `sort column` and `visible messages` have failed with -10000 since at least 2015 (Mojave through Big Sur), and still fail here. The suggested substitute is the viewer's `messages`. Sources: [Late Night Software forum](https://forum.latenightsw.com/t/struggling-to-change-the-sort-view-of-mail-using-applescript/3246), [applescript-users 2015](https://lists.apple.com/archives/applescript-users/2015/Jul/msg00038.html).
- **Attachment MIME type**: "AppleEvent handler failed" (-10000) around `mail attachments` is reported from Yosemite onward for IMAP and iCloud. I found no Exchange-specific report, but it reproduces here on Exchange. Source: [MacScripter](https://www.macscripter.net/t/get-mail-attachments-fails-with-10000-general-error-os-x-10-10-1/68463).
- **Rule scripts**: reports of handlers receiving incomplete or wrong messages (needing a delay) and of rules not firing on Sonoma. Sources: [Apple Community](https://discussions.apple.com/thread/254983536), [MacScripter](https://www.macscripter.net/t/perform-mail-action-with-messages-no-result/73012).

## Surprises

1. Exchange `source()` returned far less than `messageSize` (19 KB against 88 KB) with no multipart boundary. It may not be byte-faithful RFC 822 on Exchange; I checked one message only.
2. The Exchange account reports type `unknown`, a null `serverName` and a failing `deliveryAccount`.
3. `mimeType()` throws on Exchange attachments. amail's `read.js` already wraps it in `optional()`, so the type will come back empty there.
4. `messageId()` omits the angle brackets, which matters when building `message://` URLs or matching against In-Reply-To.
5. `Mail.signatures()` returns `null` when no signatures exist.
6. Message access is roughly ten times slower on Google than on Exchange, and `whose({messageId})` takes 1.7 s on a 13,000-message inbox.
7. There is no scripting for VIPs, smart mailboxes or conversations, and no command to apply rules on demand.
8. `accountDirectory` reveals the store path (`V10`) even though the terminal cannot list it without Full Disk Access.
9. Native reply threading is available through `reply`, but its body handling is historically unreliable, so it needs a hands-on test before it replaces the current approach.
