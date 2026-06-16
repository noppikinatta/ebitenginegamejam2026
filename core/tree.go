package core

import (
	"fmt"
	"strings"
)

// WeaponKind determines the firing pattern and stat scaling of a leaf weapon.
type WeaponKind int

const (
	KindCannon  WeaponKind = iota // balanced auto-fire
	KindShotgun                   // 3-projectile spread, short range
	KindSniper                    // single high-damage shot, very long range
)

func (k WeaponKind) String() string {
	switch k {
	case KindShotgun:
		return "Shotgun"
	case KindSniper:
		return "Sniper"
	default:
		return "Cannon"
	}
}

// TreeNode is a node in the turret wiring tree.
// Branch nodes (Weapon==nil) split incoming energy equally among their children.
// Leaf nodes (Weapon!=nil) deliver energy to their weapon.
type TreeNode struct {
	ID       int
	Name     string
	parent   *TreeNode
	Children []*TreeNode
	Weapon   *Weapon // non-nil only for leaf nodes
}

func (n *TreeNode) addChild(c *TreeNode) {
	c.parent = n
	n.Children = append(n.Children, c)
}

// TurretTree holds the generator-to-weapon wiring topology.
// Disconnecting a non-root node removes it and its entire subtree; the parent
// then redistributes its incoming energy equally among remaining children.
type TurretTree struct {
	Root        *TreeNode
	TotalEnergy float64
	nextID      int
}

// NewInitialTree creates the starting wiring tree:
//
//	Generator
//	├── Port Array
//	│   ├── Cannon α
//	│   └── Shotgun α
//	└── Starboard Array
//	    ├── Cannon β
//	    └── Sniper α
//
// Total energy = 8; each leaf gets 2 initially (all weapons are weak).
func NewInitialTree() *TurretTree {
	t := &TurretTree{TotalEnergy: 8}

	root := t.newBranch("Generator")

	left := t.newBranch("Port Array")
	root.addChild(left)
	left.addChild(t.newLeaf("Cannon α", KindCannon))
	left.addChild(t.newLeaf("Shotgun α", KindShotgun))

	right := t.newBranch("Starboard Array")
	root.addChild(right)
	right.addChild(t.newLeaf("Cannon β", KindCannon))
	right.addChild(t.newLeaf("Sniper α", KindSniper))

	t.Root = root
	t.SyncEnergy()
	return t
}

func (t *TurretTree) newBranch(name string) *TreeNode {
	n := &TreeNode{ID: t.nextID, Name: name}
	t.nextID++
	return n
}

func (t *TurretTree) newLeaf(name string, kind WeaponKind) *TreeNode {
	n := t.newBranch(name)
	n.Weapon = NewWeapon(name, 0, kind)
	return n
}

// EnergyFor returns the energy that flows into node n.
// Computed by tracing the path from root (where energy = TotalEnergy) and
// dividing equally at each branch.
func (t *TurretTree) EnergyFor(n *TreeNode) float64 {
	if n == t.Root {
		return t.TotalEnergy
	}
	parentE := t.EnergyFor(n.parent)
	return parentE / float64(len(n.parent.Children))
}

// SyncEnergy pushes computed energy into every leaf weapon.
func (t *TurretTree) SyncEnergy() {
	walkTree(t.Root, func(n *TreeNode) {
		if n.Weapon != nil {
			n.Weapon.Energy = t.EnergyFor(n)
		}
	})
}

// LeafWeapons returns all weapon instances in the tree in DFS order.
func (t *TurretTree) LeafWeapons() []*Weapon {
	var ws []*Weapon
	walkTree(t.Root, func(n *TreeNode) {
		if n.Weapon != nil {
			ws = append(ws, n.Weapon)
		}
	})
	return ws
}

// AllNodes returns all nodes (including root) in DFS order.
func (t *TurretTree) AllNodes() []*TreeNode {
	var ns []*TreeNode
	walkTree(t.Root, func(n *TreeNode) { ns = append(ns, n) })
	return ns
}

func countLeafWeapons(n *TreeNode) int {
	if n.Weapon != nil {
		return 1
	}
	total := 0
	for _, c := range n.Children {
		total += countLeafWeapons(c)
	}
	return total
}

// CanDisconnect returns true if removing this node still leaves at least one
// weapon in the tree. The root can never be disconnected.
func (t *TurretTree) CanDisconnect(n *TreeNode) bool {
	if n == t.Root {
		return false
	}
	totalLeaves := countLeafWeapons(t.Root)
	subtreeLeaves := countLeafWeapons(n)
	return totalLeaves-subtreeLeaves >= 1
}

// Disconnect removes the node with the given ID and its subtree. After removal,
// any branch nodes that have become empty are also pruned. Returns false if the
// node cannot be disconnected.
func (t *TurretTree) Disconnect(id int) bool {
	node := t.findByID(t.Root, id)
	if node == nil || !t.CanDisconnect(node) {
		return false
	}
	parent := node.parent
	for i, c := range parent.Children {
		if c == node {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			break
		}
	}
	t.pruneEmptyBranches()
	t.SyncEnergy()
	return true
}

// pruneEmptyBranches removes any non-root branch nodes that have no children
// (e.g., after all of their leaf weapons were disconnected in earlier cuts).
func (t *TurretTree) pruneEmptyBranches() {
	for {
		var empties []*TreeNode
		walkTree(t.Root, func(n *TreeNode) {
			if n != t.Root && n.Weapon == nil && len(n.Children) == 0 {
				empties = append(empties, n)
			}
		})
		if len(empties) == 0 {
			return
		}
		for _, n := range empties {
			parent := n.parent
			for i, c := range parent.Children {
				if c == n {
					parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
					break
				}
			}
		}
	}
}

func (t *TurretTree) findByID(n *TreeNode, id int) *TreeNode {
	if n.ID == id {
		return n
	}
	for _, c := range n.Children {
		if found := t.findByID(c, id); found != nil {
			return found
		}
	}
	return nil
}

func walkTree(n *TreeNode, fn func(*TreeNode)) {
	fn(n)
	for _, c := range n.Children {
		walkTree(c, fn)
	}
}

// DisconnectLabel returns a short display label for a disconnect choice.
func (t *TurretTree) DisconnectLabel(n *TreeNode) string {
	var lostNames []string
	walkTree(n, func(nd *TreeNode) {
		if nd.Weapon != nil {
			lostNames = append(lostNames, nd.Name)
		}
	})
	if len(lostNames) == 0 {
		return fmt.Sprintf("Remove %s", n.Name)
	}
	return fmt.Sprintf("Remove %s", n.Name)
}

// DisconnectDesc returns a longer description showing what is lost.
func (t *TurretTree) DisconnectDesc(n *TreeNode) string {
	var lost []string
	walkTree(n, func(nd *TreeNode) {
		if nd.Weapon != nil {
			lost = append(lost, nd.Name)
		}
	})
	if len(lost) == 0 {
		return "No weapons lost"
	}
	return fmt.Sprintf("Lose: %s — remaining weapons gain more energy", strings.Join(lost, ", "))
}
