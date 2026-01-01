# Agent Coordination Log

Purpose: shared handoff notes between agents working on this repo. Append new
entries; do not delete or overwrite prior content.

Format (append at bottom):
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
