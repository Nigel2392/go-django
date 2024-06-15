package pages

import "fmt"

type TypeName string

type Definition interface {
	// NewPage creates a new page of the type.
	NewPage() Page

	// GetPage returns the page from the database.
	// It is queried by it's ID.
	GetPage(id string) (Page, error)

	// ParentPageTypes returns the types of pages that can be parents.
	ParentPageTypes() []TypeName

	// SubpageTypes returns the types of subpages that can be created.
	SubpageTypes() []TypeName

	Query(query PageQuery) ([]Page, error)
}

type Page interface {
	ID() string
	Slug() string

	// Type hash is used to determine the type of the page
	// This should be stored in the database on the base- page object.
	// It should be a unique identifier for the type of page.
	TypeHash() TypeName

	// DBPath returns the path of the page.
	// This defines the structure in the databse.
	// It allows for pages to be in a tree structure.
	DBPath() string

	// SetDBPath sets the path of the page.
	SetDBPath(path string)

	// DBDepth returns the depth of the page in the tree.
	DBDepth() int

	// SetDBDepth sets the depth of the page in the tree.
	SetDBDepth(depth int)
}

func Subpages(p Page) ([]Page, error) {
	var typeHash = p.TypeHash()
	var definition, ok = registry.pages[typeHash]
	if !ok {
		return nil, fmt.Errorf("page type %s not registered", typeHash)
	}

	var (
		pathStartsWith = expression(QueryExpressionStartsW, "path", p.DBPath())
		depthEquals    = expression(QueryExpressionEquals, "depth", p.DBDepth()+1)
	)

	var query = Query(QueryTypeAnd, action(QueryActionSelect, nil, nil), pathStartsWith, depthEquals)
	return definition.Query(query)
}

type pageRegistry struct {
	pages map[TypeName]Definition
}

var registry = pageRegistry{
	pages: make(map[TypeName]Definition),
}
