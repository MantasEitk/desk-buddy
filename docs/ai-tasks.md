# Let an AI manage your tasks

You can ask an AI assistant to change your Desk Buddy tasks for you: *"add 'call Sam' to my tasks"*, *"tick off the invoice"*, *"what's on my list?"*. The Kindle updates within a few seconds.

## Which assistants

- **Claude Code** and **Codex** (and other AI tools that run on your Mac and can use Terminal): yes. They talk to Desk Buddy's server directly.
- **ChatGPT or Claude in the browser**: no. They run on someone else's computer and can't reach your Desk Buddy (on your Mac or your own cloud). Use the Kindle, the Mac app or the phone page instead.

## Set it up

1. Desk Buddy › **This Mac** › **AI assistants** › **Copy instructions**.
2. Paste them into your assistant's instructions file:
   - **Claude Code**: `~/.claude/CLAUDE.md` (for every project), or a project's own `CLAUDE.md`
   - **Codex**: `~/.codex/AGENTS.md`
3. Ask away.

**Your key is never in the instructions.** They tell the assistant to read the address and key from Desk Buddy's settings file on your Mac when it needs them, and never to print them. That also means they keep working when you switch between This Mac and the cloud.

## What the instructions say

For reference (the copied version has your Mac's exact path filled in):

- **Settings file:** `~/Library/Application Support/DeskBuddy/backend.json`. `mode` `"local"` → server `http://127.0.0.1:8787`; `"cloud"` → the address in `cloud`. Send `key` as an `X-Key` header, and always send a User-Agent.
- **Lists** ("profile"): `work` is the Mac list, `home` the Windows list (with one list, always `work`).

| Do | Request |
|---|---|
| See everything (with each task's `id`) | `GET /api/state` |
| Add a task | `POST /api/tasks` `{"profile": "work", "text": "Call Sam"}` (add `"parent_id": 12` for a subtask, `"notes"` for a description) |
| Edit | `PATCH /api/tasks/<id>` `{"text"?, "notes"?}` |
| Tick / untick (subtasks follow) | `POST /api/tasks/<id>/toggle` |
| Pin as "Now working on" / unpin | `POST /api/tasks/<id>/pin` / `POST /api/unpin/<profile>` |
| Move up / down | `POST /api/tasks/<id>/move/up` or `/move/down` |
| Delete (with its subtasks) | `DELETE /api/tasks/<id>` |

Example, reading the settings without printing the key:

```sh
F="$HOME/Library/Application Support/DeskBuddy/backend.json"
KEY=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['key'])" "$F")
SERVER=$(python3 -c "import json,sys;d=json.load(open(sys.argv[1]));print(d['cloud'] if d.get('mode')=='cloud' else 'http://127.0.0.1:8787')" "$F")
curl -s -A DeskBuddy -H "X-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"profile":"work","text":"Call Sam"}' "$SERVER/api/tasks"
```

## Privacy

The assistant only talks to **your own** Desk Buddy server. What it then does with your task text is up to that assistant and its company, so check their privacy terms if your tasks are sensitive. Remove the instructions from the file any time to stop it.
