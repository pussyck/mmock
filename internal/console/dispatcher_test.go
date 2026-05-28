package console

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestWriteFileAtomicReplacesContentAndCleansTempFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "stub.json")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	if err := writeFileAtomic(target, []byte("new"), 0644); err != nil {
		t.Fatalf("writeFileAtomic returned error: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("content = %q, want %q", content, "new")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".mmock-stub-") {
			t.Fatalf("temporary file was not cleaned up: %s", entry.Name())
		}
	}
}

func TestStubContentRejectsSiblingPrefixTraversal(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	siblingDir := filepath.Join(root, "config-evil")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.Mkdir(siblingDir, 0755); err != nil {
		t.Fatalf("create sibling dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siblingDir, "secret.json"), []byte("secret"), 0644); err != nil {
		t.Fatalf("write sibling file: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/stubs/../config-evil/secret.json", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	dispatcher := &Dispatcher{ConfigPath: configDir}

	if err := dispatcher.stubContentHandler(c); err != nil {
		t.Fatalf("stubContentHandler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "path_outside_config") {
		t.Fatalf("body = %q, want path_outside_config", rec.Body.String())
	}
}

func TestStubUpdateRejectsSiblingPrefixTraversal(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	siblingDir := filepath.Join(root, "config-evil")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.Mkdir(siblingDir, 0755); err != nil {
		t.Fatalf("create sibling dir: %v", err)
	}

	target := filepath.Join(siblingDir, "secret.json")
	if err := os.WriteFile(target, []byte("secret"), 0644); err != nil {
		t.Fatalf("write sibling file: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/stubs/../config-evil/secret.json", strings.NewReader(`{"content":"changed"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	dispatcher := &Dispatcher{ConfigPath: configDir}

	if err := dispatcher.stubUpdateHandler(c); err != nil {
		t.Fatalf("stubUpdateHandler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read sibling file: %v", err)
	}
	if string(content) != "secret" {
		t.Fatalf("sibling file content = %q, want unchanged secret", content)
	}
}
