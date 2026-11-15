package wiki

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ReadCorpus validates the entire source before any publication. Root metadata is not indexed.
func ReadCorpus(root string) ([]Document, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, errors.New("knowledge base must be a directory")
	}

	common, err := os.Lstat(filepath.Join(root, "common"))
	if err != nil || !common.IsDir() {
		return nil, errors.New("knowledge base requires common directory")
	}

	files := os.DirFS(root)
	documents := []Document{}
	err = fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == "." {
			return nil
		}

		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are forbidden: %s", path)
		}

		if entry.IsDir() {
			return nil
		}

		team, _, nested := strings.Cut(path, "/")
		if !nested || filepath.Ext(path) != ".md" {
			return nil
		}

		if !entry.Type().IsRegular() {
			return fmt.Errorf("not a regular file: %s", path)
		}

		raw, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}

		document := Parse(path, string(raw))
		document.Team = team
		documents = append(documents, document)

		return nil
	})

	return documents, err
}
