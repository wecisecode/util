package sortedmap

// redBlackTree is an implemantation of a Red Black Tree
type redBlackTree struct {
	compare Compare
	root    *redBlackNode
	min     *redBlackNode
	max     *redBlackNode
	size    int
}

// redBlackNode is a node of the redBlackTree
type redBlackNode struct {
	item  Item
	left  *redBlackNode
	right *redBlackNode
	red   bool
}

func NewRedBlackTree(compare Compare) *redBlackTree {
	return &redBlackTree{compare: compare}
}

// Size returns the number of nodes in the redBlackTree
func (t *redBlackTree) Size() int {
	return t.size
}

// Child returns the left or right node of the redBlackTree
func (n *redBlackNode) Child(right bool) *redBlackNode {
	if right {
		return n.right
	}
	return n.left
}

func (n *redBlackNode) setChild(right bool, node *redBlackNode) {
	if right {
		n.right = node
	} else {
		n.left = node
	}
}

// returns true if redBlackNode is red
func isRed(node *redBlackNode) bool {
	return node != nil && node.red
}

func singleRotate(oldroot *redBlackNode, dir bool) *redBlackNode {
	newroot := oldroot.Child(!dir)

	oldroot.setChild(!dir, newroot.Child(dir))
	newroot.setChild(dir, oldroot)

	oldroot.red = true
	newroot.red = false

	return newroot
}

func doubleRotate(root *redBlackNode, dir bool) *redBlackNode {
	root.setChild(!dir, singleRotate(root.Child(!dir), !dir))
	return singleRotate(root, dir)
}

func (t *redBlackTree) Min() Item {
	return t.min.item
}

func (t *redBlackTree) Max() Item {
	return t.max.item
}

// Insert inserts a value and string into the tree
// Returns true on succesful insertion, false if duplicate exists
func (t *redBlackTree) Insert(item Item) (ret bool) {
	if t.root == nil {
		t.root = &redBlackNode{item: item}
		t.min = t.root
		t.max = t.root
		ret = true
	} else {
		var head = &redBlackNode{}

		var dir = true
		var last = true

		var parent *redBlackNode  // parent
		var gparent *redBlackNode // grandparent
		var ggparent = head       // great grandparent
		var node = t.root

		ggparent.right = t.root

		for {
			if node == nil {
				// insert new node at bottom
				node = &redBlackNode{item: item, red: true}
				parent.setChild(dir, node)
				if t.compare(item, t.min.item) < 0 {
					t.min = node
				} else if t.compare(item, t.max.item) > 0 {
					t.max = node
				}
				ret = true
			} else if isRed(node.left) && isRed(node.right) {
				// flip colors
				node.red = true
				node.left.red, node.right.red = false, false
			}
			// fix red violation
			if isRed(node) && isRed(parent) {
				dir2 := ggparent.right == gparent

				if node == parent.Child(last) {
					ggparent.setChild(dir2, singleRotate(gparent, !last))
				} else {
					ggparent.setChild(dir2, doubleRotate(gparent, !last))
				}
			}

			cmp := t.compare(node.item, item)

			// stop if found
			if cmp == 0 {
				break
			}

			last = dir
			dir = cmp < 0

			// update helpers
			if gparent != nil {
				ggparent = gparent
			}
			gparent = parent
			parent = node

			node = node.Child(dir)
		}

		t.root = head.right
	}

	// make root black
	t.root.red = false

	if ret {
		t.size++
	}

	return ret
}

// Delete removes a value from the redBlackTree
// Returns true on succesful deletion, false if val is not in tree
func (t *redBlackTree) Delete(item Item) bool {
	if t.root == nil {
		return false
	}

	var head = &redBlackNode{red: true} // fake red node to push down
	var node = head
	var parent *redBlackNode  //parent
	var gparent *redBlackNode //grandparent
	var found *redBlackNode

	var dir = true

	node.right = t.root

	for node.Child(dir) != nil {
		last := dir

		// update helpers
		gparent = parent
		parent = node
		node = node.Child(dir)

		cmp := t.compare(node.item, item)

		dir = cmp < 0

		// save node if found
		if cmp == 0 {
			found = node
		}

		// pretend to push red node down
		if !isRed(node) && !isRed(node.Child(dir)) {
			if isRed(node.Child(!dir)) {
				sr := singleRotate(node, dir)
				parent.setChild(last, sr)
				parent = sr
			} else {
				sibling := parent.Child(!last)
				if sibling != nil {
					if !isRed(sibling.Child(!last)) && !isRed(sibling.Child(last)) {
						// flip colors
						parent.red = false
						sibling.red, node.red = true, true
					} else {
						dir2 := gparent.right == parent

						if isRed(sibling.Child(last)) {
							gparent.setChild(dir2, doubleRotate(parent, last))
						} else if isRed(sibling.Child(!last)) {
							gparent.setChild(dir2, singleRotate(parent, last))
						}

						gpc := gparent.Child(dir2)
						gpc.red = true
						node.red = true
						gpc.left.red, gpc.right.red = false, false
					}
				}
			}
		}
	}

	// get rid of node if we've found one
	if found != nil {
		found.item = node.item
		parent.setChild(parent.right == node, node.Child(node.left == nil))
		if found == t.min {
			t.min = parent
			if parent == head {
				t.min = head.right
			}
		} else if found == t.max {
			t.max = parent
			if parent == head {
				t.max = head.right
			}
		}
		t.size--
	}

	t.root = head.right
	if t.root != nil {
		t.root.red = false
	}

	return found != nil
}

func (t *redBlackTree) search(n *redBlackNode, item Item) (Item, bool) {
	cmp := t.compare(n.item, item)
	if cmp == 0 {
		return n.item, true
	} else if cmp < 0 {
		if n.left != nil {
			return t.search(n.left, item)
		}
	} else if n.right != nil {
		return t.search(n.right, item)
	}
	return nil, false
}

// Search searches for a value in the redBlackTree, returns the string and true
// if found or the empty string and false if val is not in the tree.
func (t *redBlackTree) Search(item Item) (Item, bool) {
	if t.root == nil {
		return nil, false
	}
	return t.search(t.root, item)
}

func (t *redBlackTree) VisitAscend(from Item, p ItemVisitor) {
	t.visitAscend(t.root, from, p)
}

func (t *redBlackTree) visitAscend(node *redBlackNode, item Item, p ItemVisitor) bool {
	if node == nil {
		return true
	}

	cmp := t.compare(node.item, item)
	// skip left branch when all its values are smaller than val
	if cmp >= 0 {
		b := t.visitAscend(node.left, item, p)
		if !b {
			return false
		}
	}

	if cmp >= 0 {
		b := p(node.item)
		if !b {
			return false
		}
	}

	b := t.visitAscend(node.right, item, p)
	if !b {
		return false
	}
	return true
}

func (t *redBlackTree) VisitDescend(to Item, p ItemVisitor) {
	t.visitAscend(t.root, to, p)
}

func (t *redBlackTree) visitDescend(node *redBlackNode, item Item, p ItemVisitor) bool {
	if node == nil {
		return true
	}

	cmp := t.compare(node.item, item)
	// skip left branch when all its values are smaller than val
	if cmp <= 0 {
		b := t.visitDescend(node.right, item, p)
		if !b {
			return false
		}
	}

	if cmp <= 0 {
		b := p(node.item)
		if !b {
			return false
		}
	}

	b := t.visitDescend(node.left, item, p)
	if !b {
		return false
	}
	return true
}
