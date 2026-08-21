package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPI   = "https://app.asana.com/api/1.0"
	docsURL      = "https://github.com/koinunopochi/asana-cli-for-agents/tree/main/docs"
	defaultLimit = 50
	maxLimit     = 100
	taskFields   = "gid,name,notes,completed,assignee,projects,memberships,custom_fields,start_on,due_on,created_at,modified_at"
)

var version = "0.0.1-dev"

type client struct {
	httpClient       *http.Client
	baseURL          string
	accessToken      string
	defaultWorkspace string
}

type options struct {
	values map[string]string
	bools  map[string]bool
	args   []string
}

type apiError struct {
	status int
	body   string
}

func (e *apiError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("asana API returned HTTP %d", e.status)
	}
	return fmt.Sprintf("asana API returned HTTP %d: %s", e.status, e.body)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		writeRootHelp(out)
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(out, version)
		return 0
	}
	if args[0] == "help" {
		if len(args) == 1 {
			writeRootHelp(out)
		} else {
			writeCommandHelp(args[1], out)
		}
		return 0
	}
	if containsHelp(args[1:]) {
		writeCommandHelp(args[0], out)
		return 0
	}

	opts, err := parseOptions(args[1:])
	if err != nil {
		return usageError(errOut, err)
	}

	cl, err := newClientFromEnv()
	if err != nil {
		return usageError(errOut, err)
	}

	switch args[0] {
	case "me":
		if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true}); err != nil {
			return usageError(errOut, err)
		}
		return executeRead(cl, opts, out, "GET", "/users/me", nil, nil)
	case "workspace":
		return runWorkspace(cl, opts, out, errOut)
	case "project":
		return runProject(cl, opts, out, errOut)
	case "section":
		return runSection(cl, opts, out, errOut)
	case "user":
		return runUser(cl, opts, out, errOut)
	case "task":
		return runTask(cl, opts, out, errOut)
	default:
		return usageError(errOut, fmt.Errorf("unknown command %q; run asana --help", args[0]))
	}
}

func newClientFromEnv() (*client, error) {
	token := os.Getenv("ASANA_ACCESS_TOKEN")
	if token == "" {
		token = os.Getenv("ASANA_PAT")
	}
	if token == "" {
		return nil, errors.New("ASANA_ACCESS_TOKEN is not set (ASANA_PAT is accepted as a compatibility alias)")
	}
	baseURL := strings.TrimRight(os.Getenv("ASANA_API_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultAPI
	}
	return &client{
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		baseURL:          baseURL,
		accessToken:      token,
		defaultWorkspace: os.Getenv("ASANA_DEFAULT_WORKSPACE_GID"),
	}, nil
}

func (c *client) request(ctx context.Context, method, path string, query url.Values, payload any) (json.RawMessage, error) {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid API endpoint: %w", err)
	}
	endpoint.RawQuery = query.Encode()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &apiError{status: resp.StatusCode, body: compactErrorBody(responseBody)}
	}
	if len(responseBody) == 0 {
		return json.RawMessage("{}"), nil
	}
	if !json.Valid(responseBody) {
		return nil, fmt.Errorf("Asana returned a non-JSON response")
	}
	return json.RawMessage(responseBody), nil
}

func compactErrorBody(body []byte) string {
	var value any
	if json.Unmarshal(body, &value) == nil {
		if encoded, err := json.Marshal(value); err == nil {
			body = encoded
		}
	}
	const max = 1000
	if len(body) > max {
		return string(body[:max]) + "..."
	}
	return strings.TrimSpace(string(body))
}

func runWorkspace(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 1 || opts.args[0] != "list" {
		return usageError(errOut, errors.New("usage: asana workspace list [--limit N] [--offset TOKEN]"))
	}
	if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true, "limit": true, "offset": true}); err != nil {
		return usageError(errOut, err)
	}
	q, err := pagingQuery(opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q.Set("opt_fields", "gid,name,is_organization")
	return executeRead(c, opts, out, "GET", "/workspaces", q, nil)
}

func runProject(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) == 0 {
		return usageError(errOut, errors.New("usage: asana project list|get <GID>"))
	}
	allowed := map[string]bool{"format": true, "pretty": true, "workspace": true, "limit": true, "offset": true}
	if opts.args[0] == "list" {
		if len(opts.args) != 1 {
			return usageError(errOut, errors.New("usage: asana project list [--workspace GID] [--limit N] [--offset TOKEN]"))
		}
		if err := requireOptions(opts, allowed); err != nil {
			return usageError(errOut, err)
		}
		workspace, err := workspaceValue(c, opts)
		if err != nil {
			return usageError(errOut, err)
		}
		q, err := pagingQuery(opts)
		if err != nil {
			return usageError(errOut, err)
		}
		q.Set("opt_fields", "gid,name,archived,public,owner,team,created_at,modified_at")
		return executeRead(c, opts, out, "GET", "/workspaces/"+escapePath(workspace)+"/projects", q, nil)
	}
	if opts.args[0] == "get" && len(opts.args) == 2 {
		if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true}); err != nil {
			return usageError(errOut, err)
		}
		q := url.Values{"opt_fields": []string{"gid,name,notes,archived,public,owner,team,workspace,created_at,modified_at"}}
		return executeRead(c, opts, out, "GET", "/projects/"+escapePath(opts.args[1]), q, nil)
	}
	return usageError(errOut, errors.New("usage: asana project list|get <GID>"))
}

func runSection(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 2 || opts.args[0] != "list" {
		return usageError(errOut, errors.New("usage: asana section list <PROJECT_GID> [--limit N] [--offset TOKEN]"))
	}
	if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true, "limit": true, "offset": true}); err != nil {
		return usageError(errOut, err)
	}
	q, err := pagingQuery(opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q.Set("opt_fields", "gid,name,project,created_at,modified_at")
	return executeRead(c, opts, out, "GET", "/projects/"+escapePath(opts.args[1])+"/sections", q, nil)
}

func runUser(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 1 || opts.args[0] != "list" {
		return usageError(errOut, errors.New("usage: asana user list [--workspace GID] [--limit N] [--offset TOKEN]"))
	}
	if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true, "workspace": true, "limit": true, "offset": true}); err != nil {
		return usageError(errOut, err)
	}
	workspace, err := workspaceValue(c, opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q, err := pagingQuery(opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q.Set("opt_fields", "gid,name,email,photo,workspaces")
	return executeRead(c, opts, out, "GET", "/workspaces/"+escapePath(workspace)+"/users", q, nil)
}

func runTask(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) == 0 {
		return usageError(errOut, errors.New("usage: asana task get|list|search|create|update|complete|comment|add-project|set-fields"))
	}
	subcommand := opts.args[0]
	switch subcommand {
	case "get":
		if len(opts.args) != 2 {
			return usageError(errOut, errors.New("usage: asana task get <GID>"))
		}
		if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true}); err != nil {
			return usageError(errOut, err)
		}
		q := url.Values{"opt_fields": []string{taskFields}}
		return executeRead(c, opts, out, "GET", "/tasks/"+escapePath(opts.args[1]), q, nil)
	case "list":
		return runTaskList(c, opts, out, errOut)
	case "search":
		return runTaskSearch(c, opts, out, errOut)
	case "create":
		return runTaskCreate(c, opts, out, errOut)
	case "update":
		return runTaskUpdate(c, opts, out, errOut)
	case "complete":
		return runTaskComplete(c, opts, out, errOut)
	case "comment":
		return runTaskComment(c, opts, out, errOut)
	case "add-project":
		return runTaskAddProject(c, opts, out, errOut)
	case "set-fields":
		return runTaskSetFields(c, opts, out, errOut)
	default:
		return usageError(errOut, fmt.Errorf("unknown task command %q", subcommand))
	}
}

func runTaskList(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 1 {
		return usageError(errOut, errors.New("usage: asana task list --project GID [--limit N] [--offset TOKEN]"))
	}
	if err := requireOptions(opts, map[string]bool{"format": true, "pretty": true, "project": true, "limit": true, "offset": true}); err != nil {
		return usageError(errOut, err)
	}
	project, err := requiredValue(opts, "project")
	if err != nil {
		return usageError(errOut, err)
	}
	q, err := pagingQuery(opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q.Set("opt_fields", taskFields)
	return executeRead(c, opts, out, "GET", "/projects/"+escapePath(project)+"/tasks", q, nil)
}

func runTaskSearch(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 1 {
		return usageError(errOut, errors.New("usage: asana task search --workspace GID [--text TEXT] [filters...]"))
	}
	allowed := map[string]bool{
		"format": true, "pretty": true, "workspace": true, "text": true, "assignee": true,
		"project": true, "section": true, "completed": true, "completed-since": true,
		"due-before": true, "due-after": true, "limit": true,
	}
	if err := requireOptions(opts, allowed); err != nil {
		return usageError(errOut, err)
	}
	workspace, err := workspaceValue(c, opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q, err := pagingQuery(opts)
	if err != nil {
		return usageError(errOut, err)
	}
	q.Set("opt_fields", taskFields)
	setQueryValue(q, opts, "text", "text")
	setQueryValue(q, opts, "assignee", "assignee.any")
	setQueryValue(q, opts, "project", "projects.any")
	setQueryValue(q, opts, "section", "sections.any")
	setQueryValue(q, opts, "completed-since", "completed_since")
	setQueryValue(q, opts, "due-before", "due_on.before")
	setQueryValue(q, opts, "due-after", "due_on.after")
	if value, ok := opts.bools["completed"]; ok {
		q.Set("completed", strconv.FormatBool(value))
	}
	return executeRead(c, opts, out, "GET", "/workspaces/"+escapePath(workspace)+"/tasks/search", q, nil)
}

func runTaskCreate(c *client, opts options, out, errOut io.Writer) int {
	allowed := map[string]bool{"confirm": true, "format": true, "pretty": true, "workspace": true, "name": true, "notes": true, "assignee": true, "start-on": true, "due-on": true, "parent": true, "project": true, "completed": true}
	if len(opts.args) != 1 || opts.args[0] != "create" {
		return usageError(errOut, errors.New("usage: asana task create --name NAME [--workspace GID] [--confirm]"))
	}
	if err := requireOptions(opts, allowed); err != nil {
		return usageError(errOut, err)
	}
	if err := requireConfirm(opts); err != nil {
		return usageError(errOut, err)
	}
	name, err := requiredValue(opts, "name")
	if err != nil {
		return usageError(errOut, err)
	}
	workspace, err := workspaceValue(c, opts)
	if err != nil {
		return usageError(errOut, err)
	}
	data := map[string]any{"name": name, "workspace": workspace}
	for option, field := range map[string]string{"notes": "notes", "assignee": "assignee", "start-on": "start_on", "due-on": "due_on", "parent": "parent"} {
		if value, ok := opts.values[option]; ok {
			data[field] = value
		}
	}
	if value, ok := opts.bools["completed"]; ok {
		data["completed"] = value
	}
	if project, ok := opts.values["project"]; ok {
		data["projects"] = []string{project}
	}
	return executeWrite(c, opts, out, "POST", "/tasks", map[string]any{"data": data})
}

func runTaskUpdate(c *client, opts options, out, errOut io.Writer) int {
	allowed := map[string]bool{"confirm": true, "format": true, "pretty": true, "name": true, "notes": true, "assignee": true, "start-on": true, "due-on": true, "completed": true}
	if len(opts.args) != 2 {
		return usageError(errOut, errors.New("usage: asana task update <GID> [fields] [--confirm]"))
	}
	if err := requireOptions(opts, allowed); err != nil {
		return usageError(errOut, err)
	}
	if err := requireConfirm(opts); err != nil {
		return usageError(errOut, err)
	}
	data := map[string]any{}
	for option, field := range map[string]string{"name": "name", "notes": "notes", "assignee": "assignee", "start-on": "start_on", "due-on": "due_on"} {
		if value, ok := opts.values[option]; ok {
			data[field] = value
		}
	}
	if value, ok := opts.bools["completed"]; ok {
		data["completed"] = value
	}
	if len(data) == 0 {
		return usageError(errOut, errors.New("at least one update field is required"))
	}
	return executeWrite(c, opts, out, "PUT", "/tasks/"+escapePath(opts.args[1]), map[string]any{"data": data})
}

func runTaskComplete(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 2 {
		return usageError(errOut, errors.New("usage: asana task complete <GID> --confirm"))
	}
	if err := requireOptions(opts, map[string]bool{"confirm": true, "format": true, "pretty": true}); err != nil {
		return usageError(errOut, err)
	}
	if err := requireConfirm(opts); err != nil {
		return usageError(errOut, err)
	}
	return executeWrite(c, opts, out, "PUT", "/tasks/"+escapePath(opts.args[1]), map[string]any{"data": map[string]any{"completed": true}})
}

func runTaskComment(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 2 {
		return usageError(errOut, errors.New("usage: asana task comment <GID> --text TEXT --confirm"))
	}
	if err := requireOptions(opts, map[string]bool{"confirm": true, "format": true, "pretty": true, "text": true}); err != nil {
		return usageError(errOut, err)
	}
	if err := requireConfirm(opts); err != nil {
		return usageError(errOut, err)
	}
	text, err := requiredValue(opts, "text")
	if err != nil {
		return usageError(errOut, err)
	}
	return executeWrite(c, opts, out, "POST", "/tasks/"+escapePath(opts.args[1])+"/stories", map[string]any{"data": map[string]any{"text": text}})
}

func runTaskAddProject(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 2 {
		return usageError(errOut, errors.New("usage: asana task add-project <TASK_GID> --project GID [--section GID] --confirm"))
	}
	if err := requireOptions(opts, map[string]bool{"confirm": true, "format": true, "pretty": true, "project": true, "section": true}); err != nil {
		return usageError(errOut, err)
	}
	if err := requireConfirm(opts); err != nil {
		return usageError(errOut, err)
	}
	project, err := requiredValue(opts, "project")
	if err != nil {
		return usageError(errOut, err)
	}
	data := map[string]any{"project": project}
	if section, ok := opts.values["section"]; ok {
		data["section"] = section
	}
	return executeWrite(c, opts, out, "POST", "/tasks/"+escapePath(opts.args[1])+"/addProject", map[string]any{"data": data})
}

func runTaskSetFields(c *client, opts options, out, errOut io.Writer) int {
	if len(opts.args) != 2 {
		return usageError(errOut, errors.New("usage: asana task set-fields <TASK_GID> --custom-fields-json JSON --confirm"))
	}
	if err := requireOptions(opts, map[string]bool{"confirm": true, "format": true, "pretty": true, "custom-fields-json": true}); err != nil {
		return usageError(errOut, err)
	}
	if err := requireConfirm(opts); err != nil {
		return usageError(errOut, err)
	}
	encoded, err := requiredValue(opts, "custom-fields-json")
	if err != nil {
		return usageError(errOut, err)
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(encoded), &fields); err != nil {
		return usageError(errOut, fmt.Errorf("--custom-fields-json must be a JSON object: %w", err))
	}
	return executeWrite(c, opts, out, "PUT", "/tasks/"+escapePath(opts.args[1]), map[string]any{"data": map[string]any{"custom_fields": fields}})
}

func executeRead(c *client, opts options, out io.Writer, method, path string, query url.Values, payload any) int {
	response, err := c.request(context.Background(), method, path, query, payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := writeJSON(out, response, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func executeWrite(c *client, opts options, out io.Writer, method, path string, payload any) int {
	response, err := c.request(context.Background(), method, path, nil, payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := writeJSON(out, response, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func parseOptions(args []string) (options, error) {
	opts := options{values: map[string]string{}, bools: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			opts.args = append(opts.args, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "--") {
			opts.args = append(opts.args, arg)
			continue
		}
		nameValue := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
		name := nameValue[0]
		if name == "" {
			return options{}, errors.New("empty option name")
		}
		if isBooleanOption(name) {
			value := true
			if len(nameValue) == 2 {
				parsed, err := strconv.ParseBool(nameValue[1])
				if err != nil {
					return options{}, fmt.Errorf("--%s expects true or false", name)
				}
				value = parsed
			}
			opts.bools[name] = value
			continue
		}
		value := ""
		if len(nameValue) == 2 {
			value = nameValue[1]
		} else {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return options{}, fmt.Errorf("--%s requires a value", name)
			}
			i++
			value = args[i]
		}
		opts.values[name] = value
	}
	return opts, nil
}

func isBooleanOption(name string) bool {
	switch name {
	case "confirm", "pretty", "completed":
		return true
	default:
		return false
	}
}

func requireOptions(opts options, allowed map[string]bool) error {
	for name := range opts.values {
		if !allowed[name] {
			return fmt.Errorf("unknown option --%s", name)
		}
	}
	for name := range opts.bools {
		if !allowed[name] {
			return fmt.Errorf("unknown option --%s", name)
		}
	}
	return nil
}

func requireConfirm(opts options) error {
	if !opts.bools["confirm"] {
		return errors.New("write operation requires --confirm")
	}
	return nil
}

func requiredValue(opts options, name string) (string, error) {
	value, ok := opts.values[name]
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("--%s is required", name)
	}
	return value, nil
}

func workspaceValue(c *client, opts options) (string, error) {
	if value, ok := opts.values["workspace"]; ok && strings.TrimSpace(value) != "" {
		return value, nil
	}
	if c.defaultWorkspace != "" {
		return c.defaultWorkspace, nil
	}
	return "", errors.New("workspace GID is required; pass --workspace or set ASANA_DEFAULT_WORKSPACE_GID")
}

func pagingQuery(opts options) (url.Values, error) {
	q := url.Values{}
	limit := defaultLimit
	if value, ok := opts.values["limit"]; ok {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > maxLimit {
			return nil, fmt.Errorf("--limit must be an integer from 1 to %d", maxLimit)
		}
		limit = parsed
	}
	q.Set("limit", strconv.Itoa(limit))
	if offset, ok := opts.values["offset"]; ok {
		q.Set("offset", offset)
	}
	return q, nil
}

func setQueryValue(q url.Values, opts options, option, parameter string) {
	if value, ok := opts.values[option]; ok {
		q.Set(parameter, value)
	}
}

func escapePath(value string) string {
	return url.PathEscape(value)
}

func writeJSON(out io.Writer, response json.RawMessage, opts options) error {
	if opts.values["format"] != "" && opts.values["format"] != "json" {
		return errors.New("only --format json is supported")
	}
	if opts.bools["pretty"] {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, response, "", "  "); err != nil {
			return fmt.Errorf("format response: %w", err)
		}
		_, err := fmt.Fprintln(out, pretty.String())
		return err
	}
	_, err := fmt.Fprintln(out, string(response))
	return err
}

func containsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func usageError(out io.Writer, err error) int {
	fmt.Fprintln(out, err)
	return 2
}

func writeRootHelp(out io.Writer) {
	fmt.Fprint(out, `Asana CLI for AI agents

Usage:
  asana <command> [options]

Authentication:
  ASANA_ACCESS_TOKEN          Asana API token (ASANA_PAT is an alias)
  ASANA_DEFAULT_WORKSPACE_GID  Default workspace for commands that need one

Read:
  me                              Show the authenticated user
  workspace list                  List accessible workspaces
  project list|get <GID>          List or inspect projects
  section list <PROJECT_GID>     List project sections
  user list                       List workspace users
  task get|list|search            Inspect or search tasks

Write (always requires --confirm):
  task create|update|complete     Create or change a task
  task comment                    Add a task comment
  task add-project                Add a task to a project or section
  task set-fields                 Set task custom fields

Output:
  JSON is the default. Add --pretty for human-readable JSON.

Meta:
  version | --version               Show the build version.

Detailed documentation:
  `+docsURL+`

Run `+"`asana <command> --help`"+` for command-specific usage.
`)
}

func writeCommandHelp(command string, out io.Writer) {
	var text string
	switch command {
	case "me":
		text = "Usage: asana me [--pretty]\n\nShow the authenticated Asana user.\n"
	case "workspace":
		text = "Usage: asana workspace list [--limit N] [--offset TOKEN]\n\nList accessible workspaces.\n"
	case "project":
		text = "Usage:\n  asana project list [--workspace GID] [--limit N]\n  asana project get <GID>\n\nList or inspect projects.\n"
	case "section":
		text = "Usage: asana section list <PROJECT_GID> [--limit N] [--offset TOKEN]\n\nList sections in a project.\n"
	case "user":
		text = "Usage: asana user list [--workspace GID] [--limit N] [--offset TOKEN]\n\nList users in a workspace.\n"
	case "task":
		text = `Usage:
  asana task get <GID>
  asana task list --project GID [--limit N]
  asana task search --workspace GID [--text TEXT] [filters...]
  asana task create --name NAME [--workspace GID] [--parent GID] [--confirm]
  asana task update <GID> [fields] --confirm
  asana task complete <GID> --confirm
  asana task comment <GID> --text TEXT --confirm
  asana task add-project <TASK_GID> --project GID [--section GID] --confirm
  asana task set-fields <TASK_GID> --custom-fields-json JSON --confirm

Read commands do not mutate Asana. Every write requires the wrapper's --confirm.
`
	default:
		writeRootHelp(out)
		return
	}
	fmt.Fprint(out, text)
	fmt.Fprintln(out, "Detailed documentation:", docsURL)
}
