package core

// Upgrade is a selectable action offered on level-up. In Phase 3 every choice
// is a tree disconnect (see World.buildDisconnectChoices); the struct is kept
// generic so the scene layer does not need to know about the tree directly.
type Upgrade struct {
	Name  string
	Desc  string
	Apply func(*World)
}
