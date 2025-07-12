package pages

import (
	"fmt"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	// _ queries.QuerySetCanBeforeExec = (*PageQuerySet)(nil)
	_ queries.QuerySetCanAfterExec                                                     = (*PageQuerySet)(nil)
	_ queries.QuerySetCanClone[*PageNode, *PageQuerySet, *queries.QuerySet[*PageNode]] = (*PageQuerySet)(nil)
)

type specificPage struct {
	ids   []int64
	pages map[int64]*Page
}

type specificPreloadInfo = orderedmap.OrderedMap[string, *specificPage]

func variableBool(b ...bool) bool {
	var v bool
	if len(b) > 0 {
		v = b[0]
	}
	return v
}

type PageQuerySet struct {
	*queries.WrappedQuerySet[*PageNode, *PageQuerySet, *queries.QuerySet[*PageNode]]
	preload *specificPreloadInfo
}

func NewPageQuerySet() *PageQuerySet {
	var pageQuerySet = &PageQuerySet{}
	pageQuerySet.WrappedQuerySet = queries.WrapQuerySet(
		queries.GetQuerySet(&PageNode{}).ForEachRow(
			pageQuerySet.forEachRow,
		),
		pageQuerySet,
	)
	return pageQuerySet
}

func (qs *PageQuerySet) CloneQuerySet(wrapped *queries.WrappedQuerySet[*PageNode, *PageQuerySet, *queries.QuerySet[*PageNode]]) *PageQuerySet {
	return &PageQuerySet{
		WrappedQuerySet: wrapped,
	}
}

// forEachRow is used to preload the page object for each row.
//
// It stores each page row in a map, keyed by the content type.
// This allows us to efficiently retrieve the page object later when
// fetching the specific page instance for a node.
func (qs *PageQuerySet) forEachRow(base *queries.QuerySet[*PageNode], row *queries.Row[*PageNode]) error {
	if qs.preload == nil {
		return nil
	}

	if row.Object.PageID == 0 || row.Object.ContentType == "" {
		logger.Warnf("page with ID %d has no content type or page ID, skipping preload", row.Object.PageID)
		return nil
	}

	var preload, exists = qs.preload.Get(
		row.Object.ContentType,
	)
	if !exists {
		preload = &specificPage{
			ids:   make([]int64, 0, 1),
			pages: make(map[int64]*Page, 1),
		}
	}

	preload.ids = append(preload.ids, row.Object.PageID)
	preload.pages[row.Object.PageID] = &row.Object.PageObject

	qs.preload.Set(
		row.Object.ContentType,
		preload,
	)

	return nil
}

func (qs *PageQuerySet) AfterExec(res any) error {
	if qs.preload == nil || qs.preload.Len() == 0 {
		return nil
	}

	var specific = qs.preload
	qs.preload = nil

	for head := specific.Front(); head != nil; head = head.Next() {
		var definition = DefinitionForType(head.Key)
		if definition == nil {
			return errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no content type definition found for %s",
				head.Key,
			))
		}

		var model = definition.Object().(Page)
		var defs = model.FieldDefs()
		var primaryField = defs.Primary()
		var rows, err = queries.GetQuerySet(model).
			Filter(fmt.Sprintf("%s__in", primaryField.Name()), head.Value.ids).
			All()
		if err != nil {
			return errors.Wrapf(err, "failed to get rows for content type %s", head.Key)
		}

		for _, row := range rows {
			var primary = row.ObjectFieldDefs.Primary()
			var pk = attrs.Get[int64](row.ObjectFieldDefs, primary.Name())
			var page, exists = head.Value.pages[pk]
			if !exists {
				return errors.New(errors.CodeNoRows, fmt.Sprintf(
					"no page found for content type %s with PK %d",
					head.Key, pk,
				))
			}

			*page = row.Object
		}
	}

	return nil
}

func (qs *PageQuerySet) Specific() *PageQuerySet {
	qs = qs.Clone()
	qs.preload = orderedmap.NewOrderedMap[string, *specificPage]()
	return qs
}

func (qs *PageQuerySet) statusFlags(statusFlags StatusFlag) expr.ClauseExpression {
	// return qs.Filter("StatusFlags__bitand", statusFlags)
	return expr.Expr("StatusFlags", expr.LOOKUP_BITAND, statusFlags)
}

func (qs *PageQuerySet) StatusFlags(statusFlags StatusFlag) *PageQuerySet {
	return qs.Filter(qs.statusFlags(statusFlags))
}

func (qs *PageQuerySet) Published() *PageQuerySet {
	return qs.Filter(qs.statusFlags(StatusFlagPublished))
}

func (qs *PageQuerySet) Unpublished() *PageQuerySet {
	return qs.Filter(qs.statusFlags(StatusFlagPublished).Not(true))
}

func (qs *PageQuerySet) Types(types ...any) *PageQuerySet {
	if len(types) == 0 {
		return qs
	}

	var typeNames = make([]string, len(types))
	for i, t := range types {
		switch v := t.(type) {
		case string:
			typeNames[i] = contenttypes.ReverseAlias(v)
		case attrs.Definer:
			typeNames[i] = contenttypes.NewContentType(v).TypeName()
		case contenttypes.ContentType:
			typeNames[i] = v.TypeName()
		default:
			panic(fmt.Errorf(
				"invalid type %T for ForTypes, expected string, attrs.Definer or contenttypes.ContentType", t,
			))
		}
	}

	return qs.Filter("ContentType__in", typeNames)
}

func (qs *PageQuerySet) Ancestors(path string, depth int64, inclusive ...bool) *PageQuerySet {
	depth++

	var incl = variableBool(inclusive...)
	var paths = make([]string, depth)
	var start = 0
	if !incl {
		start = 1
	}
	for i := start; i < int(depth); i++ {
		var path, err = ancestorPath(
			path, int64(i),
		)
		if err != nil {
			panic(errors.Wrapf(
				err, "failed to get ancestor path for %s at depth %d",
				path, i,
			))
		}
		paths[i] = path
	}

	return qs.Filter("Path__in", paths)
}

func (qs *PageQuerySet) Descendants(path string, depth int64, inclusive ...bool) *PageQuerySet {
	var incl = variableBool(inclusive...)
	var exp = expr.And(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_GT, depth),
	)

	if incl {
		exp = expr.Or(
			exp,
			expr.Expr("Path", expr.LOOKUP_EXACT, path),
		)
	}

	return qs.Filter(exp)
}

func (qs *PageQuerySet) Children(path string, depth int64) *PageQuerySet {
	return qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth+1),
	)
}

func (qs *PageQuerySet) Siblings(path string, depth int64, inclusive ...bool) *PageQuerySet {
	var incl = variableBool(inclusive...)
	var parentPath, err = ancestorPath(path, 1)
	if err != nil {
		panic(errors.Wrapf(err, "failed to get parent path for %s", path))
	}

	qs = qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, parentPath),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth),
	)

	if !incl {
		qs = qs.Filter(
			expr.Expr("Path", expr.LOOKUP_EXACT, path).Not(true),
		)
	}

	return qs
}

func (qs *PageQuerySet) AncestorOf(node *PageNode, inclusive ...bool) *PageQuerySet {
	return qs.Ancestors(node.Path, node.Depth, inclusive...)
}

func (qs *PageQuerySet) DescendantOf(node *PageNode, inclusive ...bool) *PageQuerySet {
	return qs.Descendants(node.Path, node.Depth, inclusive...)
}

func (qs *PageQuerySet) ChildrenOf(node *PageNode) *PageQuerySet {
	return qs.Children(node.Path, node.Depth)
}

func (qs *PageQuerySet) SiblingsOf(node *PageNode, inclusive ...bool) *PageQuerySet {
	return qs.Siblings(node.Path, node.Depth, inclusive...)
}

func (qs *PageQuerySet) GetChildNodes(node *PageNode, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {

	return qs.Children(node.Path, node.Depth).
		Filter("StatusFlags__bitand", statusFlags).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		queryNodes()
}

func (qs *PageQuerySet) GetDescendants(path string, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error) {

	return qs.Descendants(path, depth).
		Filter("StatusFlags__bitand", statusFlags).
		Limit(int(limit)).
		Offset(int(offset)).
		OrderBy("Path").
		queryNodes()
}

// AncestorNodes returns the ancestor nodes of the given node.
//
// The path is a PageNode.Path, the depth is the depth of the page.
func (qs *PageQuerySet) GetAncestors(p string, depth int64) ([]*PageNode, error) {
	return qs.Ancestors(p, depth).queryNodes()
}

func (qs *PageQuerySet) saveSpecific(node *PageNode, creating bool) error {

	if _, ok := node.PageObject.(*PageNode); ok || node.PageObject == nil {
		return nil
	}

	if creating || node.ContentType == "" {
		node.ContentType = contenttypes.
			NewContentType(node.PageObject).
			TypeName()
	}

	err := node.PageObject.Save(qs.Context())
	if err != nil {
		return errors.Wrap(err, "failed to save page object")
	}

	if !creating || node.PageID != 0 {
		return nil
	}

	var (
		srcDefs     = node.PageObject.FieldDefs()
		dstDefs     = node.FieldDefs()
		refField, _ = dstDefs.Field("PageID")
		srcVal, _   = srcDefs.Primary().Value()
	)

	return refField.Scan(srcVal)
}

// CreateRootNode creates a new root node.
//
// The node path must be empty.
//
// The node title must not be empty.
//
// The child node title must not be empty, if not provided the page's slug (and thus URLPath) will be based on the page's title.
//
// The node path is set to a new path part based on the number of root nodes.
func (qs *PageQuerySet) AddRoot(node *PageNode) error {

	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	previousRootNodeCount, err := qs.Filter("Depth", 0).Count()
	if err != nil {
		return err
	}

	node.Path = buildPathPart(previousRootNodeCount)
	if node.Title == "" {
		return fmt.Errorf("node title must not be empty")
	}

	node.SetUrlPath(nil)
	node.Depth = 0

	if err = qs.saveSpecific(node, true); err != nil {
		return errors.Wrap(err, "failed to save specific instance")
	}

	node.PK, err = qs.insertNode(node)
	if err != nil {
		return err
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalRootCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node: node,
	})
}

// CreateChildNode creates a new child node.
//
// The parent node path must not be empty.
//
// The child node path must be empty.
//
// The child node title must not be empty, if not provided the page's slug (and thus URLPath) will be based on the page's title.
//
// The child node path is set to a new path part based on the number of children of the parent node.
func (qs *PageQuerySet) AddChild(parent, child *PageNode) error {

	var transaction, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if child.Path != "" {
		return fmt.Errorf("child path must be empty")
	}

	child.Title = strings.TrimSpace(child.Title)
	if child.Title == "" {
		return fmt.Errorf("child title must not be empty")
	}

	child.SetUrlPath(parent)
	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.Depth = parent.Depth + 1

	if err := qs.saveSpecific(child, true); err != nil {
		return errors.Wrap(err, "failed to save specific instance")
	}

	child.PK, err = qs.insertNode(child)
	if err != nil {
		return err
	}

	parent.Numchild++
	updated, err := qs.
		ExplicitSave().
		Select("Numchild").
		Filter("PK", parent.PK).
		Update(parent)
	if err != nil {
		return err
	}

	if updated == 0 {
		return fmt.Errorf("failed to update parent node with PK %d", parent.PK)
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return err
	}

	return SignalChildCreated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node:   child,
		PageID: parent.PageID,
	})
}

// UpdateNode updates a node.
//
// This function will update the node's url path if the slug has changed.
//
// In that case, it will also update the url paths of all descendants.
func (qs *PageQuerySet) UpdateNode(node *PageNode) error {

	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if node.PK == 0 {
		return fmt.Errorf("node id must not be zero")
	}

	node.Title = strings.TrimSpace(node.Title)
	if node.Title == "" {
		return fmt.Errorf("node title must not be empty")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if err = qs.saveSpecific(node, false); err != nil {
		return errors.Wrap(err, "failed to save specific instance")
	}

	oldRecord, err := qs.GetNodeByID(node.PK)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve old record with PK %d", node.PK)
	}

	if oldRecord.Slug != node.Slug {
		var parent *PageNode

		if node.Depth > 0 {
			var parentNode, err = qs.ParentNode(node.Path, int(node.Depth))
			if err != nil {
				return errors.Wrapf(err, "failed to get parent node for node with path %s", node.Path)
			}
			parent = parentNode
		}

		node.SetUrlPath(parent)
		err = qs.updateDescendantPaths(oldRecord.UrlPath, node.UrlPath, node.Path, node.PK)
		if err != nil {
			return errors.Wrapf(err,
				"failed to update descendant paths for node with path %s and PK %d",
				node.Path, node.PK,
			)
		}
	}

	err = qs.updateNode(
		node,
	)
	if err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	if err = transaction.Commit(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalNodeUpdated.Send(&PageNodeSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node:   node,
		PageID: node.PageID,
	})
}

// DeleteRootNode deletes a root node.
func (qs *PageQuerySet) DeleteRootNode(node *PageNode) error {

	if node.Depth != 0 {
		return fmt.Errorf("node is not a root node")
	}

	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	transaction, err := qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	descendants, err := qs.GetDescendants(
		node.Path, node.Depth+1, StatusFlagNone, 0, 1000,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	var ids = make([]int64, len(descendants))
	for i, descendant := range descendants {
		if err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Ctx: qs.Context(),
			},
			Node:   descendant,
			PageID: node.PageID,
		}); err != nil {
			return err
		}
		ids[i] = descendant.PK
	}

	err = qs.deleteNodes(append(ids, node.PK))
	if err != nil {
		return errors.Wrap(err, "failed to delete nodes")
	}

	return transaction.Commit(qs.Context())
}

// DeleteNode deletes a page node.
func (qs *PageQuerySet) DeleteNode(node *PageNode) error { //, newParent *PageNode) error {
	if node.Depth == 0 {
		return qs.DeleteRootNode(node)
	}

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback(qs.Context())

	parentPath, err := ancestorPath(
		node.Path, 1,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get parent path for node with path %s", node.Path)
	}

	parent, err := qs.GetNodeByPath(
		parentPath,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get parent node for node with path %s", node.Path)
	}

	var descendants []*PageNode
	descendants, err = qs.GetDescendants(
		node.Path, node.Depth-1, StatusFlagNone, 0, 1000,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	var ids = make([]int64, len(descendants))
	for i, descendant := range descendants {
		if err = SignalNodeBeforeDelete.Send(&PageNodeSignal{
			BaseSignal: BaseSignal{
				Ctx: qs.Context(),
			},
			Node:   descendant,
			PageID: node.PageID,
		}); err != nil {
			return err
		}
		ids[i] = descendant.PK
	}

	err = qs.deleteNodes(ids)
	if err != nil {
		return errors.Wrap(err, "failed to delete descendants")
	}

	prnt, err := qs.decrementNumChild(parent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to decrement parent numchild")
	}
	*parent = *prnt

	return tx.Commit(qs.Context())
}

// MoveNode moves a node to a new parent.
//
// The node and new parent paths must not be empty or equal.
//
// The new parent must not be a descendant of the node.
//
// This function will update the url paths of all descendants, as well as the tree paths of the node and its descendants.
func (qs *PageQuerySet) MoveNode(node *PageNode, newParent *PageNode) error {

	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if newParent.Path == "" {
		return fmt.Errorf("new parent path must not be empty")
	}

	if node.Path == newParent.Path {
		return fmt.Errorf("node and new parent paths must not be the same")
	}

	if node.Depth == 0 {
		return fmt.Errorf("node is a root node")
	}

	if strings.HasPrefix(newParent.Path, node.Path) {
		return fmt.Errorf("new parent is a descendant of the node")
	}

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback(qs.Context())

	oldParentPath, err := ancestorPath(node.Path, 1)
	if err != nil {
		return errors.Wrap(err, "failed to get old parent path")
	}

	oldParent, err := qs.GetNodeByPath(oldParentPath)
	if err != nil {
		return errors.Wrap(err, "failed to get old parent node")
	}

	nodes, err := qs.GetDescendants(node.Path, node.Depth-1, StatusFlagNone, 0, 1000)
	if err != nil {
		return errors.Wrap(err, "failed to get descendants")
	}

	for _, descendant := range nodes {
		descendant := descendant
		descendant.Path = newParent.Path + descendant.Path[node.Depth*STEP_LEN:]
		descendant.Depth = (newParent.Depth + descendant.Depth + 1) - node.Depth
	}

	updated, err := qs.
		Select("Path", "Depth").
		ExplicitSave().
		BulkUpdate(nodes)

	if err != nil {
		return errors.Wrap(err, "failed to update descendants")
	}

	if updated == 0 {
		return errors.NoChanges.Wrapf("failed to update descendants for node with path %s", node.Path)
	}

	// Update url paths of descendants
	var newPath, oldPath = node.SetUrlPath(newParent)
	node.Path = newParent.Path + buildPathPart(int64(
		newParent.Numchild,
	))
	node.Depth = newParent.Depth + 1

	if err = qs.updateNode(node); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	if err = qs.updateDescendantPaths(oldPath, newPath, node.Path, node.PK); err != nil {
		return errors.Wrap(err, "failed to update descendant paths")
	}

	prnt, err := qs.incrementNumChild(newParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to increment new parent numchild")
	}
	*newParent = *prnt

	_, err = qs.decrementNumChild(oldParent.PK)
	if err != nil {
		return errors.Wrap(err, "failed to decrement old parent numchild")
	}

	if err = tx.Commit(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return SignalNodeMoved.Send(&PageMovedSignal{
		BaseSignal: BaseSignal{
			Ctx: qs.Context(),
		},
		Node:      node,
		Nodes:     nodes,
		OldParent: oldParent,
		NewParent: newParent,
	})
}

// PublishNode will set the published flag on the node
// and update it accordingly in the database.
func (qs *PageQuerySet) PublishNode(node *PageNode) error {
	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	if node.StatusFlags.Is(StatusFlagPublished) {
		return nil
	}

	node.StatusFlags |= StatusFlagPublished
	return qs.updateNodeStatusFlags(int64(StatusFlagPublished), node.PK)
}

// UnpublishNode will unset the published flag on the node
// and update it accordingly in the database.
//
// If unpublishChildren is true, it will also unpublish all descendants.
func (qs *PageQuerySet) UnpublishNode(node *PageNode, unpublishChildren bool) error {
	if node.Path == "" {
		return fmt.Errorf("node path must not be empty")
	}

	var transaction, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(qs.Context())

	if !node.StatusFlags.Is(StatusFlagPublished) {
		return nil
	}

	var xp expr.ClauseExpression = expr.Q("PK", node.PK)
	if unpublishChildren {
		xp = expr.Or(
			xp,
			expr.And(
				expr.Q("StatusFlags__bitand", int64(StatusFlagPublished)),
				expr.Q("Path__startswith", node.Path),
				expr.Q("Depth__gt", node.Depth),
			),
		)
	}

	updated, err := qs.
		ExplicitSave().
		Select("StatusFlags").
		Filter(xp).
		Update(
			&PageNode{},
			expr.Expr("StatusFlags", expr.LOOKUP_BITAND, ^int64(StatusFlagPublished)),
		)
	if err != nil {
		return errors.Wrap(err, "failed to update node status flags")
	}
	if updated == 0 {
		return errors.NoChanges.Wrapf("failed to unpublish node with PK %d", node.PK)
	}

	return transaction.Commit(qs.Context())
}

// ParentNode returns the parent node of the given node.
func (qs *PageQuerySet) ParentNode(path string, depth int) (v *PageNode, err error) {
	if depth == 0 {
		return v, ErrPageIsRoot
	}
	var parentPath string
	parentPath, err = ancestorPath(
		path, 1,
	)
	if err != nil {
		return v, err
	}
	return qs.GetNodeByPath(
		parentPath,
	)
}
