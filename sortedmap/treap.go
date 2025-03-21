package sortedmap

type Treap struct {
	compare Compare
	root    *Node
}

type Node struct {
	item     Item
	priority int
	left     *Node
	right    *Node
}

func NewTreap(c Compare) *Treap {
	return &Treap{compare: c, root: nil}
}

func (t *Treap) Min() Item {
	n := t.root
	if n == nil {
		return nil
	}
	for n.left != nil {
		n = n.left
	}
	return n.item
}

func (t *Treap) Max() Item {
	n := t.root
	if n == nil {
		return nil
	}
	for n.right != nil {
		n = n.right
	}
	return n.item
}

func (t *Treap) Get(target Item) Item {
	n := t.root
	for n != nil {
		c := t.compare(target, n.item)
		if c < 0 {
			n = n.left
		} else if c > 0 {
			n = n.right
		} else {
			return n.item
		}
	}
	return nil
}

// Note: only the priority of the first insert of an item is used.
// Priorities from future updates on already existing items are
// ignored.  To change the priority for an item, you need to do a
// Delete then an Upsert.
func (t *Treap) Upsert(item Item, itemPriority int) *Treap {
	r := t.union(t.root, &Node{item: item, priority: itemPriority})
	return &Treap{compare: t.compare, root: r}
}

func (t *Treap) union(this *Node, that *Node) *Node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	if this.priority > that.priority {
		left, middle, right := t.split(that, this.item)
		if middle == nil {
			return &Node{
				item:     this.item,
				priority: this.priority,
				left:     t.union(this.left, left),
				right:    t.union(this.right, right),
			}
		}
		return &Node{
			item:     middle.item,
			priority: this.priority,
			left:     t.union(this.left, left),
			right:    t.union(this.right, right),
		}
	}
	// We don't use middle because the "that" has precendence.
	left, _, right := t.split(this, that.item)
	return &Node{
		item:     that.item,
		priority: that.priority,
		left:     t.union(left, that.left),
		right:    t.union(right, that.right),
	}
}

// Splits a treap into two treaps based on a split item "s".
// The result tuple-3 means (left, X, right), where X is either...
// nil - meaning the item s was not in the original treap.
// non-nil - returning the Node that had item s.
// The tuple-3's left result treap has items < s,
// and the tuple-3's right result treap has items > s.
func (t *Treap) split(n *Node, s Item) (*Node, *Node, *Node) {
	if n == nil {
		return nil, nil, nil
	}
	c := t.compare(s, n.item)
	if c == 0 {
		return n.left, n, n.right
	}
	if c < 0 {
		left, middle, right := t.split(n.left, s)
		return left, middle, &Node{
			item:     n.item,
			priority: n.priority,
			left:     right,
			right:    n.right,
		}
	}
	left, middle, right := t.split(n.right, s)
	return &Node{
		item:     n.item,
		priority: n.priority,
		left:     n.left,
		right:    left,
	}, middle, right
}

func (t *Treap) Delete(target Item) *Treap {
	left, _, right := t.split(t.root, target)
	return &Treap{compare: t.compare, root: t.join(left, right)}
}

// All the items from this are < items from that.
func (t *Treap) join(this *Node, that *Node) *Node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	if this.priority > that.priority {
		return &Node{
			item:     this.item,
			priority: this.priority,
			left:     this.left,
			right:    t.join(this.right, that),
		}
	}
	return &Node{
		item:     that.item,
		priority: that.priority,
		left:     t.join(this, that.left),
		right:    that.right,
	}
}

// Visit items greater-than-or-equal to the pivot, in ascending order.
func (t *Treap) VisitAscend(pivot Item, visitor ItemVisitor) {
	t.visitAscend(t.root, pivot, visitor)
}

func (t *Treap) visitAscend(n *Node, pivot Item, visitor ItemVisitor) bool {
	if n == nil {
		return true
	}
	c := t.compare(pivot, n.item)
	if c < 0 && !t.visitAscend(n.left, pivot, visitor) {
		return false
	}
	if c <= 0 && !visitor(n.item) {
		return false
	}
	return t.visitAscend(n.right, pivot, visitor)
}

// Visit items less-than-or-equal to the pivot, in descending order.
func (t *Treap) VisitDescend(pivot Item, visitor ItemVisitor) {
	t.visitDescend(t.root, pivot, visitor)
}

func (t *Treap) visitDescend(n *Node, pivot Item, visitor ItemVisitor) bool {
	if n == nil {
		return true
	}
	c := t.compare(pivot, n.item)
	if c > 0 && !t.visitDescend(n.right, pivot, visitor) {
		return false
	}
	if c >= 0 && !visitor(n.item) {
		return false
	}
	return t.visitDescend(n.left, pivot, visitor)
}
