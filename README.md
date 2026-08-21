# asana-cli-for-agents

Asana REST API CLI for AI agents. It is a small, environment-authenticated
client with JSON output and an explicit confirmation boundary for mutations.

The CLI talks directly to the official Asana REST API. It does not require an
Asana MCP client, does not reuse MCP-only OAuth tokens, and does not persist
credentials in the repository.

## Quick start

```bash
# Inject these variables through the environment's approved secret-management
# path before starting the CLI. Do not put a token literal in shell history.
export ASANA_ACCESS_TOKEN
export ASANA_DEFAULT_WORKSPACE_GID

go build -o bin/asana .
./bin/asana --help
./bin/asana me --pretty
./bin/asana project list --pretty
./bin/asana task search --text "release" --pretty
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

## Read and write safety

Read commands are available without an extra confirmation. Task creation,
updates, completion, comments, project membership changes, and custom-field
changes require the wrapper-specific `--confirm` flag. The flag is not passed
to Asana; it is the local safety boundary.

The CLI intentionally does not provide a delete command in its initial scope.
Deletion can be added later only with an explicit safety design and tests.

## Build and test

```bash
make build
make test
make lint
```

The initial repository is private while the command surface and API behavior
are being verified. Release packaging can be added after the interface is
stable.

## Official references

- [Asana API](https://developers.asana.com/docs/api-features)
- [Authentication](https://developers.asana.com/docs/authentication)
- [Personal access token](https://developers.asana.com/docs/personal-access-token)
- [Search tasks in a workspace](https://developers.asana.com/reference/searchtasksforworkspace)
