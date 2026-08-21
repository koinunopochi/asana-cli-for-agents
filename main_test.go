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

func withEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for key, value := range values {
		t.Setenv(key, value)
	}
}
