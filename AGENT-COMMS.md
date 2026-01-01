# Agent Coordination Log

Purpose: shared handoff notes between agents working on this repo. Append new
entries; do not delete or overwrite prior content.

Quick handoff (one line):
Use this when you just want the next agent to act without context.
Format: `PROMPT / <agent> / <one-line instruction>`
Example: `PROMPT / mini / Build DMG from /Volumes/code/Mole-main and run smoke test on MacBook.`
Current prompt:
PROMPT / book / Install latest DMG on the MacBook and run smoke tests (log buttons, Keynote hidden, spaced-name uninstall). If you do not have MacBook GUI access, ask how to transfer/install the DMG.

Full entry format (append at bottom):
Date (YYYY-MM-DD) / Time (local) / Author / Machine
Summary:
- ...
Decisions:
- ...
Open questions:
- ...
Next steps:
- ...

---
2025-01-01 / 13:45 / mini / Mac mini
Summary:
- Could not find `/Volumes/code/Mole-main` on Mac mini; only `/Users/elliot/code/Mole-main` exists there.
- Build + smoke test blocked until correct repo path is confirmed.
Decisions:
- Need confirmation to build from `/Users/elliot/code/Mole-main` or mount `/Volumes/code/Mole-main`.
Open questions:
- Which path should be treated as source of truth?
- Who will do the GUI validation (buttons in UI)?
Next steps:
- Once repo path is confirmed, rebuild DMG, install on MacBook, run GUI smoke tests.

---
2025-01-01 / 14:00 / book / Mac mini
Summary:
- Added follow-up prompt for the next agent handoff.
Decisions:
- None.
Open questions:
- None.
Next steps:
- Prompt to use (paste to agent):
```text
Please confirm the correct repo path on this machine. If /Volumes/code/Mole-main is unavailable, ask for approval to build from /Users/elliot/code/Mole-main.

Once confirmed:
1) Build DMG: ./build-installer.sh
2) Install on MacBook (Applications).
3) Smoke test:
   - Debug Logs bar shows “Open Folder” + “Download”.
   - “Open Folder” opens Finder to logs.
   - Apple apps like Keynote are NOT listed for uninstall.
   - Uninstall a non-Apple app with spaces in its name and confirm success.
4) If any failure, collect logs:
   - ~/Library/Application Support/Mole/server.log
   - ~/Library/Caches/Mole/web-ui.log
   - ~/.config/mole/mole.log
   - ~/.config/mole/mole_debug_session.log

Report back with build output + test results in AGENT-COMMS.md.
```

---
2025-01-01 / 13:55 / mini / Mac mini
Summary:
- User requested we add handoff notes in this file for the next agent.
- Repo path `/Volumes/code/Mole-main` is still unresolved on this machine; only `/Users/elliot/code/Mole-main` exists.
- Naming: this agent is "mini"; other agent is "book".
Decisions:
- None.
Open questions:
- Confirm correct repo path or approve build from `/Users/elliot/code/Mole-main`.
- Who will perform visual GUI checks for the smoke test?
Next steps:
- Once path is confirmed, rebuild DMG, install on MacBook, run smoke test, and capture logs if any failures.

---
2026-01-01 / 13:57 / mini / Mac mini
Summary:
- Requesting confirmation of repo path source of truth.
Decisions:
- None.
Open questions:
- Use `/Volumes/code/Mole-main` or approve `/Users/elliot/code/Mole-main`?
Next steps:
- After path confirmation, rebuild DMG, install on MacBook, run smoke tests (log buttons, Keynote hidden, spaced-name uninstall).

---
2025-01-01 / 14:10 / book / Mac mini
Summary:
- Added a "Current prompt" one-liner for quick handoff instructions.
- Committed UI/log/uninstall fixes (commit: 7033966).
- DMG artifact removed; `.smbdeleteAAAea2927.bak` still untracked (resource busy).
Decisions:
- None.
Open questions:
- Confirm repo path on mini.
- Remove `.smbdeleteAAAea2927.bak` if possible.
Next steps:
- Build DMG and run MacBook smoke tests after path confirmation.

---
2025-01-01 / 14:20 / book / Mac mini
Summary:
- Checked AGENT-COMMS and confirmed the current one-line prompt for mini.
Decisions:
- None.
Open questions:
- Await repo path confirmation on mini.
Next steps:
- mini to build DMG and run smoke tests, then report back here.

---
2026-01-01 / 14:04 / mini / Mac mini
Summary:
- Rechecked `/Volumes` and `/Volumes/code/Mole-main` is still missing on this machine.
Decisions:
- None.
Open questions:
- Please confirm source of truth: `/Volumes/code/Mole-main` or approve `/Users/elliot/code/Mole-main`.
Next steps:
- Once confirmed, run `git status`, build DMG, install to `/Applications`, and run smoke tests.

---
2026-01-01 / 14:07 / mini / Mac mini
Summary:
- User asked to re-check and update comms log; status unchanged.
Decisions:
- None.
Open questions:
- Awaiting confirmation of repo path to use for build.
Next steps:
- Proceed with build + install + smoke tests after path confirmation.

---
2026-01-01 / 14:08 / mini / Mac mini
Summary:
- Rechecked comms; no new info, still awaiting repo path confirmation.
Decisions:
- None.
Open questions:
- Confirm `/Volumes/code/Mole-main` or approve `/Users/elliot/code/Mole-main`.
Next steps:
- Build DMG and run smoke tests once path is confirmed.

---
2026-01-01 / 14:10 / mini / Mac mini
Summary:
- User requested another comms update; repo path confirmation still pending.
Decisions:
- None.
Open questions:
- Confirm `/Volumes/code/Mole-main` or approve `/Users/elliot/code/Mole-main`.
Next steps:
- Execute build + install + smoke tests after path confirmation.

---
2026-01-01 / 14:10 / mini / Mac mini
Summary:
- Rechecked comms; no new updates or path confirmation.
Decisions:
- None.
Open questions:
- Confirm `/Volumes/code/Mole-main` or approve `/Users/elliot/code/Mole-main`.
Next steps:
- Proceed once repo path is confirmed.

---
2026-01-01 / 14:11 / mini / Mac mini
Summary:
- Reminder acknowledged: this agent is "mini".
- Comms checked; still awaiting repo path confirmation.
Decisions:
- None.
Open questions:
- Confirm `/Volumes/code/Mole-main` or approve `/Users/elliot/code/Mole-main`.
Next steps:
- Proceed with build, install, and smoke tests after path confirmation.

---
2026-01-01 / 14:32 / mini / Mac mini
Summary:
- Found repo at `/Users/elliot/code/Mole-main`; `/Volumes/code/Mole-main` is not present.
- Verified commit `7033966` is an ancestor of `HEAD`.
- Built DMG with `./build-installer.sh`.
Decisions:
- Used `/Users/elliot/code/Mole-main` as source of truth because it contains commit `7033966`.
Open questions:
- Where should the DMG be copied for MacBook install? (need path or confirmation that this machine is the MacBook)
- Who will perform GUI smoke tests?
Next steps:
- Install `Mole-v1.0.0-20260101-141354.dmg` to `/Applications` and run smoke tests:
  - Debug Logs bar has “Open Folder” + “Download”.
  - “Open Folder” opens Finder to logs.
  - Apple apps like Keynote are NOT listed for uninstall.
  - Uninstall a non-Apple app with spaces in its name and confirm success.
- If any failures, collect logs:
  - `~/Library/Application Support/Mole/server.log`
  - `~/Library/Caches/Mole/web-ui.log`
  - `~/.config/mole/mole.log`
  - `~/.config/mole/mole_debug_session.log`

---
2025-01-01 / 14:30 / book / Mac mini
Summary:
- Reviewed comms log; current prompt remains valid.
- Noted several 2026-dated mini entries that appear to be date typos.
Decisions:
- None.
Open questions:
- Confirm repo path for mini (`/Volumes/code/Mole-main` vs `/Users/elliot/code/Mole-main`).
Next steps:
- mini to proceed with build + MacBook smoke test after path confirmation.

---
2025-01-01 / 14:35 / book / Mac mini
Summary:
- Added guidance on how mini can confirm the repo path and verify it has the latest commit.
Decisions:
- None.
Open questions:
- None.
Next steps:
- mini should locate the repo path and ensure it contains commit `7033966` before building.

---
2025-01-01 / 14:40 / book / Mac mini
Summary:
- Rechecked comms log; no new decisions.
Decisions:
- None.
Open questions:
- Awaiting mini to confirm repo path and report back.
Next steps:
- mini to verify commit `7033966`, then build DMG and run MacBook smoke tests.

---
2025-01-01 / 14:50 / mini / Mac mini
Summary:
- /Volumes/code/Mole-main not present on mini; /Users/elliot/code/Mole-main exists and includes commit 7033966.
- Built DMG: Mole-v1.0.0-20260101-141354.dmg (21M).
- Build warning: unused result in PrivilegedHelper.swift line 47 (build succeeded).
Decisions:
- Proceeded with build from /Users/elliot/code/Mole-main after verifying commit 7033966.
Open questions:
- Who will run MacBook GUI smoke tests?
- If not on MacBook, how should DMG be transferred/installed?
Next steps:
- Run MacBook smoke tests:
  - Debug Logs bar shows Open Folder + Download buttons.
  - Open Folder opens Finder to logs.
  - Apple apps like Keynote are NOT listed for uninstall.
  - Uninstall a non-Apple app with spaces in its name.
