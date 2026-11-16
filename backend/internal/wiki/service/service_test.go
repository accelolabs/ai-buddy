package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	// Project packages.
	"backend/internal/users"
	"backend/internal/wiki"
)

type repositoryStub struct {
	documents []wiki.Document
	document  wiki.Document
	err       error
}

func (r *repositoryStub) List(context.Context) ([]wiki.Document, error) {
	return r.documents, r.err
}

func (r *repositoryStub) Get(context.Context, string) (wiki.Document, error) {
	return r.document, r.err
}

func TestList(t *testing.T) {
	common := wiki.Document{ID: "common"}
	own := wiki.Document{ID: "own", Team: "backend"}
	foreign := wiki.Document{ID: "foreign", Team: "frontend"}
	repo := &repositoryStub{documents: []wiki.Document{common, foreign, own}}
	service := NewService(repo, "unused")
	for _, role := range []string{users.RoleIntern, users.RoleMentor} {
		values, err := service.List(t.Context(), users.User{Role: role, Team: "backend"})
		if err != nil || !reflect.DeepEqual(values, []wiki.Document{common, own}) {
			t.Fatalf("documents: %+v; %v", values, err)
		}
	}
	repo.documents = []wiki.Document{foreign}
	if values, err := service.List(t.Context(), users.User{Team: "backend"}); err != nil ||
		values == nil ||
		len(values) != 0 {
		t.Fatalf("empty: %v; %v", values, err)
	}
	repo.err = errors.New("database error")
	if _, err := service.List(t.Context(), users.User{}); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	root := t.TempDir()
	content := []byte("# Статья\r\n\n**Markdown**\t\n\x00")
	if err := os.WriteFile(filepath.Join(root, "article.md"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(root, "escape.md")); err != nil {
		t.Fatal(err)
	}
	repo := &repositoryStub{document: wiki.Document{ID: "article", Path: "article.md", Team: "backend"}}
	service := NewService(repo, root)
	user := users.User{Team: "backend"}
	for _, team := range []string{"", "backend"} {
		repo.document.Team = team
		got, err := service.Get(t.Context(), user, "article")
		if err != nil || !reflect.DeepEqual(got, content) {
			t.Fatalf("markdown: %q; %v", got, err)
		}
	}
	for _, path := range []string{"../secret.md", outside, "escape.md"} {
		repo.document.Path = path
		if got, err := service.Get(t.Context(), user, "article"); !errors.Is(err, ErrReadFile) || len(got) != 0 {
			t.Fatalf("escape %q: %q, %v", path, got, err)
		}
	}
	repo.document.Path = "missing.md"
	if _, err := service.Get(t.Context(), user, "article"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	service.root = filepath.Join(root, "missing-root")
	if _, err := service.Get(t.Context(), user, "article"); !errors.Is(err, ErrReadFile) || errors.Is(err, ErrNotFound) {
		t.Fatalf("root: %v", err)
	}
	repo.document.Team = "foreign"
	if _, err := service.Get(t.Context(), user, "article"); !errors.Is(err, ErrNotFound) || errors.Is(err, ErrReadFile) {
		t.Fatalf("access before open: %v", err)
	}
	repo.err = fmt.Errorf("get: %w", sql.ErrNoRows)
	if _, err := service.Get(t.Context(), user, "article"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	repo.err = errors.New("metadata failure")
	if _, err := service.Get(t.Context(), user, "article"); !errors.Is(err, repo.err) || errors.Is(err, ErrReadFile) {
		t.Fatal(err)
	}
}
