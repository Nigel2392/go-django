package pages

/*
	Fixtree:
		- Retrieve all pages from the database

		- Sort the pages by path

		- Find parents and children

			* Example:
				if strings.HasPrefix(child.Path, parent.Path) && child.Depth == parent.Depth + 1 {
					// child is a direct child of parent
				}

		- Create a tree structure from the sorted pages
		  	* Example:
				type PageNode struct {
					Node *models.PageNode
					Children []*PageNode
				}

		- Loop over the tree and fix the path of each node
			* This can be done by sorting the children and then simply using buildPathPart(forLoopIndex) to generate the correct path for each child





*/
