# Agent instructions

Use the CLI help as the source of truth for invocation:

1. Run `asana --help` to see the command groups and documentation entry point.
2. Run `asana <command> --help` before using a command.
3. Read `docs/README.md` and the linked page when the task needs authentication,
   search limitations, output, or write safety details.

This CLI uses the Asana REST API directly. It does not connect to an Asana MCP
server and it does not store tokens in this repository.

- Read operations may run without `--confirm`.
- Every write operation requires the wrapper's explicit `--confirm`.
- Never print or persist `ASANA_ACCESS_TOKEN` or `ASANA_PAT`.
- Do not assume that an empty search result proves that a task does not exist.
  Search indexing is eventually consistent and permissions affect visibility.
