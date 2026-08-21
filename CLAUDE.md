# Asana CLI: Claude Code entry

Use the CLI help first, then the linked documentation:

```bash
asana --help
asana <command> --help
```

The CLI reads `ASANA_ACCESS_TOKEN` (with `ASANA_PAT` as a compatibility alias)
from the process environment. Do not ask the user to paste a token into chat,
do not add tokens to command arguments, and do not use an Asana MCP tool as a
fallback. Add `--confirm` only after the requested write target and mutation
are explicit.

@AGENTS.md
