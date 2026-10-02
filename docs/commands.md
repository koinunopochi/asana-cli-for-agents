# Commands

Run the command's own help before using it. JSON is the default output format;
add `--pretty` when a human-readable response is useful.

## Me

```bash
asana me
asana me --pretty
```

## Workspace

```bash
asana workspace list
asana workspace list --limit 100 --pretty
```

## Project and section

```bash
asana project list --workspace WORKSPACE_GID
asana project get PROJECT_GID
asana section list PROJECT_GID
```

`--workspace` may be omitted when `ASANA_DEFAULT_WORKSPACE_GID` is set.

## Team and custom field reads

```bash
asana team list --workspace WORKSPACE_GID
asana field list --workspace WORKSPACE_GID
asana project fields PROJECT_GID
```

`team list` finds the team GID required to create a project in an
organization. `field list` finds existing workspace custom fields.
`project fields` lists the custom field settings on a project; each setting GID
can be used as an insert position for `project add-field`.

## Project and section writes

All commands in this section require `--confirm`.

```bash
asana project create --workspace WORKSPACE_GID --team TEAM_GID --name "Project" --default-view board --confirm
asana section create PROJECT_GID --name "Doing" --confirm
asana section create PROJECT_GID --name "Review" --insert-after SECTION_GID --confirm
asana project add-field PROJECT_GID --field FIELD_GID --important --confirm
asana project add-field PROJECT_GID --field-json '{"workspace":"WORKSPACE_GID","name":"Priority","resource_subtype":"enum","enum_options":[{"name":"High"},{"name":"Low"}]}' --confirm
```

- Board-view columns are sections; add them with `section create`.
- List-view columns are custom fields; add them with `project add-field`.
- `--team` is required when the workspace is an organization.
- `--field-json` creates a new custom field. The CLI sends the JSON object
  unchanged as `custom_field`; include the required `workspace` and
  `resource_subtype` yourself.
- `--insert-before` and `--insert-after` are mutually exclusive. For sections
  they take a section GID; for `add-field` they take a custom field setting GID.

The CLI has no project, section, or custom field deletion command. State the
target and intended structure before adding `--confirm`.

## User reads

```bash
asana user list --workspace WORKSPACE_GID
```

## Task reads

```bash
asana task get TASK_GID
asana task list --project PROJECT_GID
asana task list --project PROJECT_GID --limit 100 --pretty
```

## Task search

```bash
asana task search --workspace WORKSPACE_GID --text "release"
asana task search --workspace WORKSPACE_GID --assignee me --completed=false
asana task search --workspace WORKSPACE_GID --project PROJECT_GID --limit 100
```

The workspace task search endpoint can require a qualifying Asana plan, and
Asana documents that search indexing is eventually consistent. An empty or
stale result is not proof that a task does not exist; use `task get` when a GID
is known.

## Task writes

All commands in this section require `--confirm`.

```bash
asana task create --workspace WORKSPACE_GID --name "Task title" --confirm
asana task create --workspace WORKSPACE_GID --name "Subtask" --parent PARENT_TASK_GID --confirm
asana task create --workspace WORKSPACE_GID --name "Completed record" --completed --confirm
asana task update TASK_GID --notes "Updated notes" --confirm
asana task update TASK_GID --start-on 2026-08-22 --due-on 2026-08-29 --confirm
asana task complete TASK_GID --confirm
asana task comment TASK_GID --text "Comment" --confirm
asana task add-project TASK_GID --project PROJECT_GID --section SECTION_GID --confirm
asana task set-fields TASK_GID --custom-fields-json '{"FIELD_GID":"VALUE"}' --confirm
```

The initial CLI has no task deletion command. The caller should inspect the
target with `task get` before a mutation and should state the intended target,
field changes, and side effects before adding `--confirm`.

## Pagination and limits

List commands accept `--limit` from 1 to 100 and an optional `--offset` token.
The CLI does not silently fetch an unbounded result set. Search has additional
filters; use `asana task search --help` for the current command surface.

## Official references

- [Task API reference](https://developers.asana.com/reference/tasks)
- [Create a project](https://developers.asana.com/reference/createproject)
- [Create a section in a project](https://developers.asana.com/reference/createsectionforproject)
- [Add a custom field to a project](https://developers.asana.com/reference/addcustomfieldsettingforproject)
- [Search tasks in a workspace](https://developers.asana.com/reference/searchtasksforworkspace)
- [Dates and times](https://developers.asana.com/docs/dates-and-times)
