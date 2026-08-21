# asana-cli-for-agents

Asana REST API CLI for AI agents. It is a small, environment-authenticated
client with JSON output and an explicit confirmation boundary for mutations.

The CLI talks directly to the official Asana REST API. It does not require an
Asana MCP client, does not reuse MCP-only OAuth tokens, and does not persist
credentials in the repository.

## Quick start

```bash
# Download the archive for your platform from GitHub Releases, then place the
# `asana` executable somewhere on PATH.
./asana --help

# Inject these variables through the environment's approved secret-management
# path before starting the CLI. Do not put a token literal in shell history.
export ASANA_ACCESS_TOKEN
export ASANA_DEFAULT_WORKSPACE_GID

asana me --pretty
asana project list --pretty
asana task search --text "release" --pretty
```

For normal operation, inject the environment variables through the approved
secret-management path for the environment. Do not commit a plain `.env`, put
tokens in shell history, or pass tokens as command arguments.

## Documentation

The executable help is the first routing step:

```bash
asana --help
asana <command> --help
```

Detailed operator and agent guidance lives in [docs/README.md](docs/README.md).

- [Authentication](docs/authentication.md)
- [Commands](docs/commands.md)
- [Output and errors](docs/output.md)
- [Releases and downloads](docs/releases.md)

## Read and write safety

Read commands are available without an extra confirmation. Task creation,
updates, completion, comments, project membership changes, and custom-field
changes require the wrapper-specific `--confirm` flag. The flag is not passed
to Asana; it is the local safety boundary.

The CLI intentionally does not provide a delete command in its initial scope.
Deletion can be added later only with an explicit safety design and tests.

## Downloads

Published releases include ready-to-run archives for Linux, macOS, and Windows
on amd64 and arm64. Download the archive for your platform from this
repository's [GitHub Releases](https://github.com/koinunopochi/asana-cli-for-agents/releases)
page; you do not need Go or a local build. Each archive includes the `asana`
binary, `LICENSE`, `NOTICE`, `README.md`, and detailed `docs/`. Verify the
download with `SHA256SUMS`.

The release workflow runs when a `v*` tag is pushed. See
[docs/releases.md](docs/releases.md) for the maintainer procedure.

## Development

```bash
make build
make test
make lint
```

`make build` is for local development. End users should use the published
release archive instead.

## Official references

- [Asana API](https://developers.asana.com/docs/api-features)
- [Authentication](https://developers.asana.com/docs/authentication)
- [Personal access token](https://developers.asana.com/docs/personal-access-token)
- [Search tasks in a workspace](https://developers.asana.com/reference/searchtasksforworkspace)
