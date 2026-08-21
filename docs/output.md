# Output and errors

## JSON output

The CLI prints the official Asana JSON response to stdout. Use `--pretty` for
indentation. Keep stdout available for machine processing and use stderr for
usage or transport errors.

```bash
asana task get TASK_GID > task.json
asana task search --workspace WORKSPACE_GID --text "release" --pretty
```

The CLI does not redact ordinary task content because that is the requested API
result. AI callers should summarize only the fields needed for the request and
avoid copying unrelated task data into chat or documents.

## Exit codes

- `0`: successful API request or help output
- `1`: network, authentication, or Asana API failure
- `2`: invalid CLI input or a missing write confirmation

Error output includes the HTTP status when available, never the bearer token.

## Search caveat

Asana's workspace task search has plan and indexing constraints. Search results
can be delayed after a write and can differ between repeated requests. When a
search result is used to decide whether to create a task, verify a known GID or
perform a narrower read before writing.
