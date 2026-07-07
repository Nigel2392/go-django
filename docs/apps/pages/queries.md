# Queries and other database operations

The Pages app offers functions to manage the tree structure and content of pages via the `PageQuerySet`.

You can construct a `PageQuerySet` using `pages.NewPageQuerySet()`.

Below are the tree management methods available on `PageQuerySet`:

- **FixTree(ctx context.Context) error**  
  Scans the database for tree inconsistencies and repairs them. Provided globally as `pages.FixTree(ctx)`.

- **AddRoot(node *PageNode) error**  
  Adds a new root node to the tree. The node's path must be empty, and a title is required.

- **AddChildren(parent *PageNode, children ...*PageNode) error**  
  Adds new child nodes under an existing parent node. The child nodes' paths are generated based on the parent’s URL path.

- **UpdateNode(node *PageNode) error**  
  Updates a node, recalculating its URL path if the slug has changed and updating all descendant paths.

- **Delete(nodes ...*PageNode) (int64, error)**  
  Deletes the specified nodes from the tree and maintains tree integrity.

- **MoveNode(node *PageNode, newParent *PageNode) error**  
  Moves a node to a new parent, updating the URL paths and tree structure accordingly.

- **PublishNode(node *PageNode) error**  
  Sets the published flag on the node and updates the database record.

- **UnpublishNode(node *PageNode, unpublishChildren bool) error**  
  Unpublishes a node (and optionally its descendants).

- **ParentNode(path string, depth int) (*PageNode, error)**  
  Retrieves the parent node based on the current node’s path.

- **AllNodes() ([]*PageNode, error)**  
  Retrieves all nodes based on the current queryset filters.

- **GetNodeByID(id int64) (*PageNode, error)**
  Retrieves a node by its ID.

- **GetNodeByPath(path string) (*PageNode, error)**
  Retrieves a node by its full path.

- **GetNodeBySlug(slug string, depth int64, path string) (*PageNode, error)**
  Retrieves a node by its slug, depth and parent path.
