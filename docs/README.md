# Asana CLI operator guide

This directory contains the details that are intentionally not duplicated in
the executable help. Start with `asana --help`, choose a command, then read the
relevant section here.

## Scope

The CLI is a thin REST API client for AI-managed Asana operations. It handles:

- environment-based authentication;
- bounded reads with JSON output;
- task, project, section, workspace, and user lookup;
- task creation options for parent tasks, dates, assignees, and completed records;
- explicit confirmation for task mutations.

It does not manage developer apps, issue tokens, workspace permissions, or
Asana MCP connections.

## Routing

| Need | First command | Details |
|---|---|---|
| Current user | `asana me` | [commands.md](commands.md#me) |
| Workspaces | `asana workspace list` | [commands.md](commands.md#workspace) |
| Projects and sections | `asana project` / `asana section` | [commands.md](commands.md#project-and-section) |
| Task details and lists | `asana task get` / `asana task list` | [commands.md](commands.md#task-reads) |
| Task search | `asana task search` | [commands.md](commands.md#task-search) |
| Task mutation | `asana task create` / `update` / `complete` | [commands.md](commands.md#task-writes) |

## Official references

- [Asana API](https://developers.asana.com/docs/api-features)
- [Authentication](https://developers.asana.com/docs/authentication)
- [API rate limits](https://developers.asana.com/docs/rate-limits)
