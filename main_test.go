package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeUsesEnvironmentTokenAndPrintsJSON(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		if r.Method != http.MethodGet || r.URL.Path != "/users/me" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"gid":"123","name":"Test User"}}`)
	}))
	defer server.Close()

	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN":          "test-token",
		"ASANA_PAT":                   "",
		"ASANA_API_BASE_URL":          server.URL,
		"ASANA_DEFAULT_WORKSPACE_GID": "",
	})

	var out, errOut strings.Builder
	if code := run([]string{"me"}, &out, &errOut); code != 0 {
		t.Fatalf("run returned %d: %s", code, errOut.String())
	}
	if gotAuthorization != "Bearer test-token" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
	if !strings.Contains(out.String(), `"gid":"123"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if strings.Contains(out.String(), "test-token") {
		t.Fatal("token leaked to stdout")
	}
}

func TestWriteRequiresConfirmBeforeRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN": "test-token",
		"ASANA_API_BASE_URL": server.URL,
	})

	var out, errOut strings.Builder
	if code := run([]string{"task", "complete", "123"}, &out, &errOut); code != 2 {
		t.Fatalf("run returned %d, want 2; stderr=%s", code, errOut.String())
	}
	if called {
		t.Fatal("write request was sent without --confirm")
	}
	if !strings.Contains(errOut.String(), "--confirm") {
		t.Fatalf("missing confirmation error: %s", errOut.String())
	}
}

func TestTaskCreateSendsExpectedPayload(t *testing.T) {
	var requestMethod, requestPath, requestAuthorization string
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMethod = r.Method
		requestPath = r.URL.Path
		requestAuthorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"gid":"task-1","name":"New task"}}`)
	}))
	defer server.Close()

	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN": "test-token",
		"ASANA_API_BASE_URL": server.URL,
	})

	var out, errOut strings.Builder
	args := []string{"task", "create", "--workspace", "workspace-1", "--name", "New task", "--notes", "Details", "--confirm"}
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("run returned %d: %s", code, errOut.String())
	}
	if requestMethod != http.MethodPost || requestPath != "/tasks" {
		t.Fatalf("request = %s %s", requestMethod, requestPath)
	}
	if requestAuthorization != "Bearer test-token" {
		t.Fatalf("authorization = %q", requestAuthorization)
	}
	data, ok := requestBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("request data = %#v", requestBody["data"])
	}
	if data["name"] != "New task" || data["workspace"] != "workspace-1" || data["notes"] != "Details" {
		t.Fatalf("unexpected request data: %#v", data)
	}
}

func TestTaskCreateSupportsCompletedParentAndStartDate(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"gid":"task-2"}}`)
	}))
	defer server.Close()

	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN": "test-token",
		"ASANA_API_BASE_URL": server.URL,
	})

	var out, errOut strings.Builder
	args := []string{"task", "create", "--workspace", "workspace-1", "--name", "Done task", "--parent", "parent-1", "--start-on", "2026-08-22", "--completed", "--confirm"}
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("run returned %d: %s", code, errOut.String())
	}
	data, ok := requestBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("request data = %#v", requestBody["data"])
	}
	if data["completed"] != true || data["parent"] != "parent-1" || data["start_on"] != "2026-08-22" {
		t.Fatalf("unexpected request data: %#v", data)
	}
}

func TestTaskSearchBuildsWorkspaceQuery(t *testing.T) {
	var gotQuery string
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()

	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN": "test-token",
		"ASANA_API_BASE_URL": server.URL,
	})

	var out, errOut strings.Builder
	args := []string{"task", "search", "--workspace", "workspace-1", "--text", "release", "--completed=false", "--limit", "10"}
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("run returned %d: %s", code, errOut.String())
	}
	if gotPath != "/workspaces/workspace-1/tasks/search" {
		t.Fatalf("path = %q", gotPath)
	}
	for _, want := range []string{"completed=false", "limit=10", "text=release"} {
		if !strings.Contains(gotQuery, want) {
			t.Fatalf("query %q does not contain %q", gotQuery, want)
		}
	}
}

func TestMissingTokenDoesNotMakeRequest(t *testing.T) {
	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN": "",
		"ASANA_PAT":          "",
		"ASANA_API_BASE_URL": "",
	})

	var out, errOut strings.Builder
	if code := run([]string{"me"}, &out, &errOut); code != 2 {
		t.Fatalf("run returned %d, want 2; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "ASANA_ACCESS_TOKEN") {
		t.Fatalf("missing token error: %s", errOut.String())
	}
}

func TestHelpDoesNotRequireAuthentication(t *testing.T) {
	withEnv(t, map[string]string{
		"ASANA_ACCESS_TOKEN": "",
		"ASANA_PAT":          "",
		"ASANA_API_BASE_URL": "",
	})

	var out, errOut strings.Builder
	if code := run([]string{"task", "--help"}, &out, &errOut); code != 0 {
		t.Fatalf("run returned %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "task search") || !strings.Contains(out.String(), "--confirm") {
		t.Fatalf("unexpected help: %s", out.String())
	}
}

func TestNewWritesRequireConfirmBeforeRequest(t *testing.T) {
	commands := [][]string{
		{"project", "create", "--workspace", "workspace-1", "--name", "Project"},
		{"project", "add-field", "project-1", "--field", "field-1"},
		{"section", "create", "project-1", "--name", "Doing"},
	}
	for _, args := range commands {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		withEnv(t, map[string]string{
			"ASANA_ACCESS_TOKEN": "test-token",
			"ASANA_API_BASE_URL": server.URL,
		})

		var out, errOut strings.Builder
		if code := run(args, &out, &errOut); code != 2 {
			t.Fatalf("%v returned %d, want 2; stderr=%s", args, code, errOut.String())
		}
		server.Close()
		if called {
			t.Fatalf("%v sent a write request without --confirm", args)
		}
		if !strings.Contains(errOut.String(), "--confirm") {
			t.Fatalf("%v missing confirmation error: %s", args, errOut.String())
		}
	}
}

func TestNewWritesSendExpectedPayload(t *testing.T) {
	tests := []struct {
		args []string
		path string
		data map[string]any
	}{
		{
			args: []string{"project", "create", "--workspace", "workspace-1", "--team", "team-1", "--name", "Project", "--default-view", "board", "--confirm"},
			path: "/projects",
			data: map[string]any{"workspace": "workspace-1", "team": "team-1", "name": "Project", "default_view": "board"},
		},
		{
			args: []string{"section", "create", "project-1", "--name", "Doing", "--insert-after", "section-1", "--confirm"},
			path: "/projects/project-1/sections",
			data: map[string]any{"name": "Doing", "insert_after": "section-1"},
		},
		{
			args: []string{"project", "add-field", "project-1", "--field", "field-1", "--important", "--confirm"},
			path: "/projects/project-1/addCustomFieldSetting",
			data: map[string]any{"custom_field": "field-1", "is_important": true},
		},
		{
			args: []string{"project", "add-field", "project-1", "--field-json", `{"name":"Priority","resource_subtype":"text"}`, "--confirm"},
			path: "/projects/project-1/addCustomFieldSetting",
			data: map[string]any{"custom_field": map[string]any{"name": "Priority", "resource_subtype": "text"}},
		},
	}
	for _, tt := range tests {
		var requestMethod, requestPath string
		var requestBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestMethod = r.Method
			requestPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":{"gid":"created-1"}}`)
		}))
		withEnv(t, map[string]string{
			"ASANA_ACCESS_TOKEN": "test-token",
			"ASANA_API_BASE_URL": server.URL,
		})

		var out, errOut strings.Builder
		code := run(tt.args, &out, &errOut)
		server.Close()
		if code != 0 {
			t.Fatalf("%v returned %d: %s", tt.args, code, errOut.String())
		}
		if requestMethod != http.MethodPost || requestPath != tt.path {
			t.Fatalf("%v request = %s %s", tt.args, requestMethod, requestPath)
		}
		got, _ := json.Marshal(requestBody["data"])
		want, _ := json.Marshal(tt.data)
		if string(got) != string(want) {
			t.Fatalf("%v data = %s, want %s", tt.args, got, want)
		}
	}
}

func TestExclusiveOptionsRejectedBeforeRequest(t *testing.T) {
	tests := [][]string{
		{"section", "create", "project-1", "--name", "Doing", "--insert-before", "a", "--insert-after", "b", "--confirm"},
		{"project", "add-field", "project-1", "--field", "field-1", "--field-json", `{}`, "--confirm"},
		{"project", "add-field", "project-1", "--confirm"},
	}
	for _, args := range tests {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		withEnv(t, map[string]string{
			"ASANA_ACCESS_TOKEN": "test-token",
			"ASANA_API_BASE_URL": server.URL,
		})

		var out, errOut strings.Builder
		code := run(args, &out, &errOut)
		server.Close()
		if code != 2 || called {
			t.Fatalf("%v returned %d, called=%v; stderr=%s", args, code, called, errOut.String())
		}
	}
}

func TestWorkspaceListsUseWorkspacePath(t *testing.T) {
	tests := map[string]string{"team": "/workspaces/workspace-1/teams", "field": "/workspaces/workspace-1/custom_fields"}
	for command, wantPath := range tests {
		var gotPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":[]}`)
		}))
		withEnv(t, map[string]string{
			"ASANA_ACCESS_TOKEN":          "test-token",
			"ASANA_API_BASE_URL":          server.URL,
			"ASANA_DEFAULT_WORKSPACE_GID": "workspace-1",
		})

		var out, errOut strings.Builder
		code := run([]string{command, "list"}, &out, &errOut)
		server.Close()
		if code != 0 || gotPath != wantPath {
			t.Fatalf("%s list returned %d, path %q; stderr=%s", command, code, gotPath, errOut.String())
		}
	}
}

func withEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for key, value := range values {
		t.Setenv(key, value)
	}
}
