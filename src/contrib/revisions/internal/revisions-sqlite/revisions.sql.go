package revisions_sqlite

import (
	"context"

	"github.com/Nigel2392/go-django/src/contrib/revisions/internal/revisions_db"
)

const deleteRevision = `-- name: DeleteRevision :exec
DELETE FROM Revision
WHERE id = ?1
`

func (q *Queries) DeleteRevision(ctx context.Context, id int64) error {
	_, err := q.exec(ctx, q.deleteRevisionStmt, deleteRevision, id)
	return err
}

const getRevisionByID = `-- name: GetRevisionByID :one
SELECT   id, object_id, content_type, data, created_at
FROM     Revision
WHERE    id = ?1
`

func (q *Queries) GetRevisionByID(ctx context.Context, id int64) (revisions_db.Revision, error) {
	row := q.queryRow(ctx, q.getRevisionByIDStmt, getRevisionByID, id)
	var i revisions_db.Revision
	err := row.Scan(
		&i.ID,
		&i.ObjectID,
		&i.ContentType,
		&i.Data,
		&i.CreatedAt,
	)
	return i, err
}

const getRevisionsByObjectID = `-- name: GetRevisionsByObjectID :many
SELECT   id, object_id, content_type, data, created_at
FROM     Revision
WHERE    object_id = ?1
AND      content_type = ?2
ORDER BY created_at DESC
LIMIT    ?3
OFFSET   ?4
`

func (q *Queries) GetRevisionsByObjectID(ctx context.Context, objectID string, contentType string, limit int32, offset int32) ([]revisions_db.Revision, error) {
	rows, err := q.query(ctx, q.getRevisionsByObjectIDStmt, getRevisionsByObjectID,
		objectID,
		contentType,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []revisions_db.Revision
	for rows.Next() {
		var i revisions_db.Revision
		if err := rows.Scan(
			&i.ID,
			&i.ObjectID,
			&i.ContentType,
			&i.Data,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertRevision = `-- name: InsertRevision :execlastid
INSERT INTO Revision (
    object_id,
    content_type,
    data
) VALUES (
    ?1,
    ?2,
    ?3
)
`

func (q *Queries) InsertRevision(ctx context.Context, objectID string, contentType string, data string) (int64, error) {
	result, err := q.exec(ctx, q.insertRevisionStmt, insertRevision, objectID, contentType, data)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const listRevisions = `-- name: ListRevisions :many
SELECT   id, object_id, content_type, data, created_at
FROM     Revision
ORDER BY created_at DESC
LIMIT    ?
OFFSET   ?
`

func (q *Queries) ListRevisions(ctx context.Context, limit int32, offset int32) ([]revisions_db.Revision, error) {
	rows, err := q.query(ctx, q.listRevisionsStmt, listRevisions, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []revisions_db.Revision
	for rows.Next() {
		var i revisions_db.Revision
		if err := rows.Scan(
			&i.ID,
			&i.ObjectID,
			&i.ContentType,
			&i.Data,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateRevision = `-- name: UpdateRevision :exec
UPDATE Revision
SET object_id = ?1,
    content_type = ?2,
    data = ?3
WHERE id = ?4
`

func (q *Queries) UpdateRevision(ctx context.Context, objectID string, contentType string, data string, iD int64) error {
	_, err := q.exec(ctx, q.updateRevisionStmt, updateRevision,
		objectID,
		contentType,
		data,
		iD,
	)
	return err
}
