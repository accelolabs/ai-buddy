package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	// Project packages.
	"backend/internal/users"
	"backend/internal/users/middleware"
	"backend/internal/wiki"
)

type readerStub struct {
	documents []wiki.Document
	document  wiki.Document
	err       error
	id        string
	calls     int
}

func (s *readerStub) List(_ context.Context) ([]wiki.Document, error) {
	s.calls++

	return s.documents, s.err
}

func (s *readerStub) Get(_ context.Context, id string) (wiki.Document, error) {
	s.calls++
	s.id = id

	return s.document, s.err
}

type userStub struct {
	role string
}

func (s userStub) Get(context.Context, string) (users.User, error) {
	return users.User{
		ID:   "selected",
		Team: "backend",
		Role: s.role,
	}, nil
}

func request(
	t *testing.T,
	stub *readerStub,
	root, target, role string,
	cookie bool,
) *httptest.ResponseRecorder {
	t.Helper()
	h := NewHandler(stub, root)
	mux := http.NewServeMux()
	selected := middleware.Middleware(userStub{role})
	mux.Handle("GET /api/wiki", selected(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/wiki/{id}", selected(http.HandlerFunc(h.Get)))
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie {
		req.AddCookie(&http.Cookie{
			Name:  "demo_user",
			Value: "selected",
		})
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	return w
}

func TestList(t *testing.T) {
	for _, tc := range []struct {
		name      string
		documents []wiki.Document
		err       error
		status    int
		body      string
	}{
		{
			name:   "empty",
			status: 200,
			body:   "[]\n",
		},
		{
			name: "metadata",
			documents: []wiki.Document{{
				ID:    "id",
				Path:  "guide.md",
				Title: "Статья",
			}},
			status: 200,
			body:   "[{\"id\":\"id\",\"team\":\"\",\"title\":\"Статья\"}]\n",
		},
		{
			name:   "database failure",
			err:    errors.New("private database error"),
			status: 500,
			body:   "failed to list articles\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &readerStub{
				documents: tc.documents,
				err:       tc.err,
			}
			w := request(t, stub, t.TempDir(), "/api/wiki", users.RoleIntern, true)
			if w.Code != tc.status || w.Body.String() != tc.body {
				t.Fatalf("response %d %q", w.Code, w.Body.String())
			}
		})
	}
}

func TestGetMarkdown(t *testing.T) {
	root := t.TempDir()
	content := "# Полная статья\n\n**Markdown**\n\n```go\nfmt.Println(1)\n```\n" + strings.Repeat(
		"Текст\n",
		10000,
	)
	if err := os.WriteFile(filepath.Join(root, "guide.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, role := range []string{users.RoleIntern, users.RoleMentor} {
		t.Run(
			role,
			func(t *testing.T) {
				stub := &readerStub{document: wiki.Document{
					ID:   "article",
					Path: "guide.md",
				}}
				w := request(t, stub, root, "/api/wiki/article", role, true)
				if w.Code != 200 || w.Body.String() != content ||
					w.Header().Get("Content-Type") != "text/markdown; charset=utf-8" ||
					stub.id != "article" {
					t.Fatalf("response %d; id %q", w.Code, stub.id)
				}
			},
		)
	}
}

func TestGetErrors(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(root, "link.md")); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, path string
		err        error
		status     int
	}{
		{
			name:   "missing",
			err:    sql.ErrNoRows,
			status: 404,
		},
		{
			name:   "database error",
			err:    errors.New("private error"),
			status: 500,
		},
		{
			name:   "missing file",
			path:   "absent.md",
			status: 404,
		},
		{
			name:   "path escape",
			path:   outside,
			status: 500,
		},
		{
			name:   "symlink escape",
			path:   "link.md",
			status: 500,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &readerStub{
				document: wiki.Document{Path: tc.path},
				err:      tc.err,
			}
			w := request(t, stub, root, "/api/wiki/article", users.RoleIntern, true)
			if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") ||
				strings.Contains(
					w.Body.String(),
					"private",
				) {
				t.Fatalf("response %d %q", w.Code, w.Body.String())
			}
		})
	}
}

func TestRequiresUser(t *testing.T) {
	for _, target := range []string{"/api/wiki", "/api/wiki/article"} {
		stub := &readerStub{}
		w := request(t, stub, t.TempDir(), target, users.RoleIntern, false)
		if w.Code != 401 || stub.calls != 0 {
			t.Fatalf("response %d, repository calls %d", w.Code, stub.calls)
		}
	}
}

func TestAccessByTeam(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "article.md"), []byte("# Статья"), 0o600); err != nil {
		t.Fatal(err)
	}

	common := wiki.Document{
		ID:    "common",
		Path:  "article.md",
		Title: "Общая",
	}
	own := wiki.Document{
		ID:    "own",
		Path:  "article.md",
		Team:  "backend",
		Title: "Своя",
	}
	foreign := wiki.Document{
		ID:    "foreign",
		Path:  "article.md",
		Team:  "platform",
		Title: "Чужая",
	}

	for _, role := range []string{users.RoleIntern, users.RoleMentor} {
		t.Run(role, func(t *testing.T) {
			stub := &readerStub{documents: []wiki.Document{common, foreign, own}}
			w := request(t, stub, root, "/api/wiki", role, true)

			var got []documentResponse
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}

			want := []documentResponse{
				{
					ID:    common.ID,
					Team:  common.Team,
					Title: common.Title,
				},
				{
					ID:    own.ID,
					Team:  own.Team,
					Title: own.Title,
				},
			}
			if w.Code != 200 || !reflect.DeepEqual(got, want) {
				t.Fatalf("list response %d %s", w.Code, w.Body.String())
			}

			stub = &readerStub{documents: []wiki.Document{foreign}}
			w = request(t, stub, root, "/api/wiki", role, true)
			if w.Code != 200 || w.Body.String() != "[]\n" {
				t.Fatalf("foreign-only list: %d %q", w.Code, w.Body.String())
			}

			for _, doc := range []wiki.Document{common, own, foreign} {
				stub = &readerStub{document: doc}
				w = request(t, stub, root, "/api/wiki/"+doc.ID, role, true)
				wantStatus := 200
				if doc.Team == "platform" {
					wantStatus = 404
				}

				if w.Code != wantStatus {
					t.Fatalf("Get(%q): %d, want %d", doc.ID, w.Code, wantStatus)
				}
			}
			// The root does not exist: denial must happen before any file access.
			stub = &readerStub{document: foreign}
			w = request(t, stub, filepath.Join(root, "absent"), "/api/wiki/foreign", role, true)
			if w.Code != 404 {
				t.Fatalf("foreign article accessed files: status %d", w.Code)
			}
		})
	}
}
