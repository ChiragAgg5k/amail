# Comparable tools survey

Gathered on 2026-11-10 while charting the v1.0 map. Links were collected by a research agent and not individually re-verified.

# Landscape for `amail` (as of 2026-11-10)

The "ride on Apple Mail" niche is already crowded: I found about 25 tools, most created in 2026. No one else serves a pure-Automation, single-binary CLI that also handles Exchange bodies Mail has not cached and truly threaded reply drafts.

**How this was checked.** Apple Mail tools: I read each README and issue tracker through the GitHub API today. Server-direct tools and the fragility section come from two sub-agents citing primary URLs; I did not re-open every one of those links. Feature cells for the classic clients (mutt, aerc, alpine, isync and similar) are general knowledge, not re-checked. Anything marked "inferred" is my reasoning.

**Name collision.** [KorigamiK/amail](https://github.com/KorigamiK/amail) is a different tool with the same name and binary, with a Homebrew tap (`korigamik/tap/amail`), created 2026-10-19. Your repo was created today.

## Comparison table

"Locked tenant" means third-party OAuth consent is blocked and IMAP, basic auth and app passwords are off. R read, S search, Se send, Re reply, A attachments, F flag, M move, D delete.

| Tool | Auth | Works in locked tenant | JSON | Features | Stars; last release or commit |
|---|---|---|---|---|---|
| **amail** (yours) | none; Automation only | yes | yes | R, S (subject/sender, one mailbox), Se, Re (unthreaded), A | 0; 2026-11-10 |
| [0xSMW/mail.app-cli](https://github.com/0xSMW/mail.app-cli) (Go CLI + TUI) | none; Full Disk Access, Automation fallback | yes | yes | all, plus rules, smart mailboxes, signatures, VIP, export, bulk, unsubscribe | 3; v3.2.2, 2026-10-31 |
| [intelligrit/mail-app-cli](https://github.com/intelligrit/mail-app-cli) (Go CLI + Emacs client) | none; Automation | yes | yes | R S Se A F M D; no drafts | 21; 2026-10-26 |
| [KorigamiK/amail](https://github.com/KorigamiK/amail) (Bun CLI) | none; Full Disk Access + Automation | yes | yes | R S A F M, thread, draft, raw SQL; cannot send | 0; v0.1.1, 2026-10-29 |
| [zydtiger/mailctl](https://github.com/zydtiger/mailctl) (Rust CLI) | none; Full Disk Access + Automation (+ Accessibility) | yes | yes | R S Se Re F M D, Send Later, Outlook meetings; no attachments | 0; v0.3.0, 2026-11-07 |
| [joargp/amcli](https://github.com/joargp/amcli) (Node CLI) | none; Full Disk Access | yes | yes | R S A, raw .eml, digest-confirmed send, native threaded reply | 1; 2026-08-25 |
| [furedea/apple-mail-cli](https://github.com/furedea/apple-mail-cli) (Rust CLI) | none; Automation only | yes | yes | R S, mark read, move (preview, then `--execute`) | 0; 2026-11-04 |
| [patrickfreyer/apple-mail-mcp](https://github.com/patrickfreyer/apple-mail-mcp) (MCP, 23 tools) | none; Automation | yes | MCP | all, plus forward, drafts, HTML .eml drafts, analytics | 250; v3.3.0, 2026-11-03 |
| [sweetrb/apple-mail-mcp](https://github.com/sweetrb/apple-mail-mcp) (MCP) | none; optional IMAP/SMTP app password | yes, AppleScript path only | MCP | all, plus rules, templates | 108; v2.20.55, 2026-11-04 |
| [s-morgan-jeffries/apple-mail-fast-mcp](https://github.com/s-morgan-jeffries/apple-mail-fast-mcp) (MCP, 25 tools) | none; optional IMAP app password | yes, AppleScript path only | MCP | all, plus rules and mailbox create/rename/delete | 105; v0.11.0, 2026-09-05 |
| [imdinu/apple-mail-mcp](https://github.com/imdinu/apple-mail-mcp) (MCP + CLI, read-only) | none; Full Disk Access | yes | yes | R, full-body search, A | 73; v0.5.0, 2026-09-14 |
| [parasxos/apple-mail-mcp](https://github.com/parasxos/apple-mail-mcp) (MCP, 21 tools) | none locally; Graph device code, SMTP for extras | local part yes; Graph extras no (inferred) | MCP | all, plus scheduling and planned triage | 36; v1.8.3, 2026-10-18 |
| [PsychQuant/che-apple-mail-mcp](https://github.com/PsychQuant/che-apple-mail-mcp) (Swift MCP, 59 tools) | none; Full Disk Access + Automation + Accessibility | yes | MCP | all, plus rules, VIP, signatures, redirect, Markdown export | 11; v4.0.0, 2026-10-29 |
| [jx-grxf/mailidx](https://github.com/jx-grxf/mailidx) (Rust MCP) | none; Full Disk Access + Automation + Accessibility | yes | MCP | all, plus .eml export, `wait_for`, one-time codes, categories | 2; v0.3.1, 2026-11-08 |
| [supermemoryai/apple-mcp](https://github.com/supermemoryai/apple-mcp) (bundle) | none; Automation | yes | MCP | Se, S, unread count | 3112; stale since 2025-08-11 |
| [Raycast Apple Mail extension](https://github.com/raycast/extensions/tree/main/extensions/mail) | none | yes | AI tools only | R S Se Re A M D, one-time-code paste | 2026-10-22 |
| [himalaya](https://github.com/pimalaya/himalaya) | token from a broker (ortie) or password; IMAP, JMAP, Gmail, Graph | no, unless the client id is approved | yes, plus `himalaya mcp` | all | 7621; v2.2.1, 2026-10-02 |
| aerc, neomutt, mutt, meli, alpine | password or bring-your-own OAuth client | no | no | all | aerc 0.22.0 (2026-07-28); neomutt 20260616; mutt 2.4.3; meli v0.9.0 (2026-11-01); alpine 2.26 (2022) |
| mbsync, OfflineIMAP3, msmtp, notmuch, mblaze, s-nail | IMAP/SMTP credentials, or local only | no | notmuch only | sync, send or local index | all active in 2026 except s-nail stable (2024) |
| [gogcli](https://github.com/openclaw/gogcli), [gws](https://github.com/googleworkspace/cli), [gmailctl](https://github.com/mbrt/gmailctl) | own Google OAuth client | no | yes | Gmail API (gmailctl: filters only) | 8570, 32222, 2212 |
| [CLI for Microsoft 365](https://github.com/pnp/cli-microsoft365), [msgraph-cli](https://github.com/microsoftgraph/msgraph-cli) (archived), [Softeria ms-365-mcp](https://github.com/Softeria/ms-365-mcp-server), [ryaker/outlook-mcp](https://github.com/ryaker/outlook-mcp) | Entra app, device code | no | yes | Graph mail | 1479, 132, 1124, 454 |
| [yusufaltunbicak/outlook-cli](https://github.com/yusufaltunbicak/outlook-cli) | captures Outlook Web tokens with Playwright | probably, while web access works | yes | all | 31; 2026-10-09 |
| [fastmail-cli](https://github.com/radiosilence/fastmail-cli), [imap-mcp-server](https://github.com/nikolausm/imap-mcp-server) | JMAP token; IMAP password | not applicable; no | yes | R S Se and more | 80, 129 |
| [AgentMail](https://docs.agentmail.to/welcome), KeyID, Goshen | API key | not applicable: they create new inboxes | hosted | own inboxes only | agentmail-mcp 72 |

Smaller Apple Mail tools I also saw: [smarzola/apple-mail-mcp](https://github.com/smarzola/apple-mail-mcp) (Rust CLI + MCP, Automation only), [NLINEA/apple-mail-cli](https://github.com/NLINEA/apple-mail-cli) (campaign checks), [macos-cli-tools/apple-mail-cli](https://github.com/macos-cli-tools/apple-mail-cli) (Bash, 33 commands), [rusty_apple_mail_mcp](https://github.com/like-a-freedom/rusty_apple_mail_mcp) (read-only), MrGo2/icloud-mcp, AirMCP, orchard-mcp. No maintained Alfred workflow turned up.

## A. Closest overlaps

- **0xSMW/mail.app-cli** (3 stars, 2026-10-31) is the closest: also Go, also a CLI, also JSON.
  - It has and you lack: cross-mailbox search from the index, a TUI, flag/move/archive/delete with `--dry-run` and `--verify`, drafts, signatures, rules, smart mailboxes, VIP, export, unsubscribe scan, a `doctor` command, and Exchange-specific handling.
  - You have and it lacks: no cgo, and no Full Disk Access needed for full function. Its README says cross-mailbox search "is refused" without it.
- **intelligrit/mail-app-cli** (21 stars, 2026-10-26) is pure Automation like yours.
  - It has: flag, move, archive, delete, cross-mailbox search, `In-Reply-To`/`References` in output, an Emacs client.
  - You have: draft-by-default and reply. Its [issue #3](https://github.com/intelligrit/mail-app-cli/issues/3) asks for drafts and is still open.
- **KorigamiK/amail** (0 stars, 2026-10-29) has index search across all accounts, thread view, mark/move/archive and a `sql` command. It deliberately cannot send, and needs Full Disk Access.
- **zydtiger/mailctl** (0 stars, 2026-11-07) has Send Later, forward, trash, flag colours and a self-attributed permission model (it re-executes itself so macOS checks the binary, not the terminal). It cannot save or attach files; you can.
- **joargp/amcli** (1 star, 2026-08-25) has native replies that keep `In-Reply-To`, raw .eml output and digest-confirmed sends. It has no flag or move.
- **furedea** and **smarzola** share your Automation-only stance but do less.
- **The MCP servers** (patrickfreyer 250, sweetrb 108, fast-mcp 105) are supersets in features. They are MCP-first, and the two that are fast on large mailboxes get there with IMAP app passwords, which a locked tenant forbids.

## B. Repeated, still-unserved requests

- **Threaded reply saved as a draft.** [patrickfreyer #56](https://github.com/patrickfreyer/apple-mail-mcp/issues/56) (open): "reply to a specific message and save the result to Drafts instead of sending". The broader redesign is [#80](https://github.com/patrickfreyer/apple-mail-mcp/issues/80), also open. Your reply has the same gap.
- **Sent body arrives as a quote.** Script-built messages are wrapped in `<blockquote type="cite">` (Apple bug FB11734014): [patrickfreyer #88](https://github.com/patrickfreyer/apple-mail-mcp/issues/88) (open), [fast-mcp #322](https://github.com/s-morgan-jeffries/apple-mail-fast-mcp/issues/322), [Apple forum thread](https://developer.apple.com/forums/thread/738842). Others work around it with SMTP or Graph, which needs credentials. Test whether your sends are affected.
- **Search on large mailboxes without IMAP.** [sweetrb #342](https://github.com/sweetrb/apple-mail-mcp/issues/342) and [#349](https://github.com/sweetrb/apple-mail-mcp/issues/349): "users who can't configure IMAP (managed Exchange accounts that block app passwords, for example)". Related hangs: [patrickfreyer #41](https://github.com/patrickfreyer/apple-mail-mcp/issues/41), [fast-mcp #415](https://github.com/s-morgan-jeffries/apple-mail-fast-mcp/issues/415).
- **Exchange bodies missing on disk.** Direct-store readers cannot see them: [parasxos #1](https://github.com/parasxos/apple-mail-mcp/issues/1), [che #299](https://github.com/PsychQuant/che-apple-mail-mcp/issues/299). parasxos fills the gap through Graph with the Microsoft Graph Command Line Tools client id, which needs consent (inferred from its source).
- **Wrong sender on multi-account setups.** [0xSMW #22](https://github.com/0xSMW/mail.app-cli/issues/22): mail meant for Exchange "went out from my personal Gmail". Also a no-send switch, [0xSMW #29](https://github.com/0xSMW/mail.app-cli/issues/29).
- **Ids shifting after writes.** [sweetrb #152](https://github.com/sweetrb/apple-mail-mcp/issues/152): a batch delete hit messages that were never passed.
- **Maintenance worry on the biggest project.** [patrickfreyer #105](https://github.com/patrickfreyer/apple-mail-mcp/issues/105), "Add more maintainers?".
- **Locked Microsoft 365 in direct clients.**
  - [himalaya #710](https://github.com/pimalaya/himalaya/issues/710): "My organization is not revealing the Azure client id"; the Thunderbird id did not work.
  - [himalaya #675](https://github.com/pimalaya/himalaya/issues/675): the maintainer says "there is not that much Himalaya can do".
  - [mutt #385](https://gitlab.com/muttmua/mutt/-/issues/385) and [#407](https://gitlab.com/muttmua/mutt/-/issues/407) are open since 2022.
  - [Softeria #443](https://github.com/Softeria/ms-365-mcp-server/issues/443): device code blocked by a managed-device policy.
  - Microsoft's side: basic auth has been off for IMAP/POP/EWS since October 2022, which also kills app passwords ([doc](https://learn.microsoft.com/en-us/exchange/clients-and-mobile-in-exchange-online/deprecation-of-basic-authentication-exchange-online)). SMTP AUTH basic goes off by default at the end of December 2026 ([timeline](https://techcommunity.microsoft.com/blog/exchange/updated-exchange-online-smtp-auth-basic-authentication-deprecation-timeline/4489835)).

## C. Use cases that need (or strongly favour) riding on Apple Mail

| Use case | Already done by |
|---|---|
| Locked Exchange/university mailbox: read, search, send | Every Apple Mail tool; none of the server-direct ones except the Outlook Web scraper |
| Read Exchange bodies Mail has not cached, with no credentials | Open gap: store readers fail, parasxos needs Graph. A scripting path can ask Mail to fetch (inferred; verify) |
| Many accounts, zero credential setup | All Apple Mail tools |
| Agent acts on mail without holding credentials, draft-first | amail, KorigamiK (never sends), amcli, patrickfreyer `--read-only`, parasxos read-only mode |
| Threaded reply draft with quoted original and signature | mailidx and che (via Accessibility paste), amcli (headers but no quote); nothing clean with Automation alone |
| Mail rules: list, create, apply | che, fast-mcp, sweetrb, 0xSMW |
| VIPs, smart mailboxes, signatures | 0xSMW (all three), che (VIP, signatures); nobody composes with a per-account signature reliably |
| Flag colours | Many |
| Apple Intelligence categories | mailidx only |
| Full-body search of the local index | imdinu, parasxos, mailidx, che; all need Full Disk Access |
| Attachments already downloaded | Most |
| Export .eml | mailidx, amcli, mailctl (raw source), che |
| Mail merge through an org account | Only NLINEA (experimental campaign checks); effectively open |
| Wait for a message, extract a one-time code | mailidx `wait_for`, Raycast paste-code; no CLI |
| Send Later | mailctl (Accessibility), parasxos (own queue or Graph) |
| Unsubscribe scan | 0xSMW, mailidx |
| Calendar invite handling | mailidx reads; mailctl creates via Outlook |
| S/MIME signed or encrypted send through Mail's keychain identity | Nobody; 0xSMW lists signing as unsupported |
| `message://` deep links | Raycast, patrickfreyer |

Least-served for a CLI: mail merge, wait/watch, threaded reply drafts, uncached Exchange bodies, and "Automation only, no Full Disk Access" at usable speed.

## D. Fragility the roadmap must respect

- **Automation permission.** The grant belongs to the hosting app (the terminal or agent host), not the binary. Denial is error -1743. Unattended runs (ssh, launchd, cron) get no prompt and time out ([Scripting OS X](https://scriptingosx.com/2020/09/avoiding-applescript-security-and-privacy-requests/), [Apple DTS](https://developer.apple.com/forums/thread/751802)). The error text is localised ([sweetrb #218](https://github.com/sweetrb/apple-mail-mcp/issues/218)).
- **Full Disk Access.** It cannot be prompted for or granted in code. On this Mac, `ls ~/Library/Mail` returned "Operation not permitted". Tools routinely tell users to grant it to the wrong app ([imdinu #124](https://github.com/imdinu/apple-mail-mcp/issues/124)).
- **Store layout.** V10 has been used from macOS 13 through 26 per the [SpamSieve manual](https://c-command.com/spamsieve/help/how-can-i-rebuild-apple); a V11 claim elsewhere is unverified. The schema is undocumented. Known traps: a hardcoded V10 ([imdinu #57](https://github.com/imdinu/apple-mail-mcp/issues/57)), Gmail labels in a separate table, `.partial.emlx`, and Exchange pruning (about 27% of bodies missing in one report).
- **Scripting limits.**
  - Mail handles one Apple event at a time, so parallel callers cascade into timeouts ([patrickfreyer #93](https://github.com/patrickfreyer/apple-mail-mcp/issues/93)).
  - `whose` scans on large Gmail or Exchange mailboxes hang Mail's UI.
  - `html content` is documented as doing nothing.
  - Reply content cannot be read, and setting it can be silently ignored ([sweetrb #7](https://github.com/sweetrb/apple-mail-mcp/issues/7)).
  - Mailbox names are localised and NFD-normalised.
  - Exchange keeps a sent copy in Drafts ([0xSMW #27](https://github.com/0xSMW/mail.app-cli/issues/27)).
  - Mail silently falls back to the default account when the sender does not match.
- **Ids.** The numeric id is the index row id. It changes when a message moves, and draft ids drift within seconds ([che #405](https://github.com/PsychQuant/che-apple-mail-mcp/issues/405)). Your README's "stable ID" claim needs that caveat; the RFC Message-ID is the better cross-check.
- **macOS 26.** 26.1 briefly returned ids wrapped in angle brackets, reported fixed in 26.2 ([MacScripter](https://www.macscripter.net/t/scripting-changes-or-lack-thereof-in-macos-tahoe/77173/33)). POP account scripting was reported broken ([SpamSieve notes](https://mjtsai.com/blog/2025/09/18/spamsieve-3-2/)). The dictionary has no category or summary terms.
- **Sandbox.** Mail is sandboxed with write access to Downloads and user-selected files (checked locally with codesign). Saving or attaching outside those may fail; untested on 26.
- **JXA.** Apple's release notes stop at OS X 10.11. No Apple statement deprecates Mail scripting; it looks frozen (inferred). Mail's App Intents on this Mac include send, reply, draft, archive and update actions, a possible supported fallback, but search is not exposed.
