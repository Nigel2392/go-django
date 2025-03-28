package page_models

const (
	FieldPK               = "id"
	FieldTitle            = "title"
	FieldPath             = "path"
	FieldDepth            = "depth"
	FieldNumchild         = "numchild"
	FieldUrlPath          = "url_path"
	FieldSlug             = "slug"
	FieldStatusFlags      = "status_flags"
	FieldPageID           = "page_id"
	FieldContentType      = "content_type"
	FieldLatestRevisionID = "latest_revision_id"
	FieldCreatedAt        = "created_at"
	FieldUpdatedAt        = "updated_at"
)

var ValidFields = []string{
	FieldPK,
	FieldTitle,
	FieldPath,
	FieldDepth,
	FieldNumchild,
	FieldUrlPath,
	FieldSlug,
	FieldStatusFlags,
	FieldPageID,
	FieldContentType,
	FieldLatestRevisionID,
	FieldCreatedAt,
	FieldUpdatedAt,
}

var _fieldMap = func() map[string]struct{} {
	m := make(map[string]struct{}, len(ValidFields))
	for _, f := range ValidFields {
		m[f] = struct{}{}
	}
	return m
}()

func IsValidField(f string) bool {
	_, ok := _fieldMap[f]
	return ok
}
