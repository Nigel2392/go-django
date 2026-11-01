package treenode

// todo: implement page- like generic tree structure

type TreeNode struct {
	NodePath     string `json:"node_path" attrs:"blank"`
	NodeDepth    int64  `json:"node_depth" attrs:"blank"`
	NodeNumchild int64  `json:"node_numchild" attrs:"blank"`
}
