# Authentication

## Environment variables

The CLI reads credentials from the process environment. It does not read a
token file and does not implement a second credential store.

| Variable | Required | Purpose |
|---|---:|---|
| `ASANA_ACCESS_TOKEN` | yes | Asana REST API bearer token; a PAT is the initial recommended value |
| `ASANA_PAT` | no | Compatibility alias used only when `ASANA_ACCESS_TOKEN` is absent |
| `ASANA_DEFAULT_WORKSPACE_GID` | no | Default workspace for commands that need one |
| `ASANA_API_BASE_URL` | no | API base override for controlled tests; defaults to the official API endpoint |

Keep the values in the environment's approved secret-management path. The
repository contains only variable names and examples with placeholder values.

## Token choices

Asana documents PAT as the quickest option for a script or a single-user
integration. PAT-authenticated changes are attributed to the user who created
the token. If a separate automation identity is required, use the appropriate
Service Account or regular OAuth design instead of sharing a personal token.

The token created for an Asana MCP app is not a replacement for an API token:
Asana documents MCP app tokens as usable only with the MCP server. This CLI
therefore uses a normal REST API token.

## Safety rules

- Never print the value of `ASANA_ACCESS_TOKEN` or `ASANA_PAT`.
- Never place a token in an argument, URL, log, README, test fixture, or issue.
- Never dump the complete environment to diagnose authentication.
- When authentication fails, report the variable name and HTTP status only.
- Do not treat an authenticated token as permission to perform an unrequested
  mutation.

## Official references

- [Asana authentication](https://developers.asana.com/docs/authentication)
- [Personal access token](https://developers.asana.com/docs/personal-access-token)
- [OAuth](https://developers.asana.com/docs/oauth)
- [Integrating with Asana's MCP Server](https://developers.asana.com/docs/integrating-with-asanas-mcp-server)
