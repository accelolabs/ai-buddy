package postgres

import (
	"context"
	"database/sql"

	"ai-buddy/internal/wiki"
	"ai-buddy/internal/wiki/indexer"
)

func (s *session) Publish(ctx context.Context, changes indexer.Changes) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, id := range changes.Deleted {
		if _, err := tx.ExecContext(ctx, "DELETE FROM documents WHERE id=$1", id); err != nil {
			return err
		}
	}
	for _, document := range changes.Documents {
		if err := replaceDocument(ctx, tx, document); err != nil {
			return err
		}
	}

	const query = `
  INSERT INTO wiki_state(singleton,fingerprint)
  VALUES(true,$1)
  ON CONFLICT(singleton) DO UPDATE SET fingerprint=excluded.fingerprint
 `
	if _, err := tx.ExecContext(ctx, query, changes.Fingerprint); err != nil {
		return err
	}

	return tx.Commit()
}

func replaceDocument(ctx context.Context, tx *sql.Tx, document wiki.Document) error {
	const query = `
  INSERT INTO documents(id,path,team,title,markdown,hash)
  VALUES($1,$2,$3,$4,$5,$6)
 `
	if _, err := tx.ExecContext(ctx, "DELETE FROM documents WHERE id=$1", document.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx, query,
		document.ID,
		document.Path,
		document.Team,
		document.Title,
		document.Markdown,
		document.Hash,
	); err != nil {
		return err
	}

	for ordinal, section := range document.Sections {
		const query = `
   INSERT INTO sections(document_id,ordinal,title,anchor,text)
   VALUES($1,$2,$3,$4,$5)
  `
		if _, err := tx.ExecContext(
			ctx, query,
			document.ID,
			ordinal,
			section.Title,
			section.Anchor,
			section.Text,
		); err != nil {
			return err
		}
	}
	for ordinal, chunk := range document.Chunks {
		const query = `
   INSERT INTO chunks(document_id,section,ordinal,text,embedding)
   VALUES($1,$2,$3,$4,$5::vector)
  `
		if _, err := tx.ExecContext(
			ctx, query,
			document.ID,
			chunk.Section,
			ordinal,
			chunk.Text,
			vector(chunk.Embedding),
		); err != nil {
			return err
		}
	}

	return nil
}
