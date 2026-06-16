package core

import "math"

// Upgrade is one selectable reward offered on level-up. For Phase 2 these are
// simple stat boosts; Phase 3 will replace this menu with the wiring-tree
// (Disconnect) screen that redistributes generator energy.
type Upgrade struct {
	Name string
	Desc string
	// Apply mutates the world to grant the upgrade.
	Apply func(*World)
}

// upgradeCatalog is the pool the level-up screen draws choices from.
func upgradeCatalog() []Upgrade {
	return []Upgrade{
		{
			Name: "Overcharge",
			Desc: "+1 energy to all weapons",
			Apply: func(w *World) {
				for _, weapon := range w.Player.Weapons {
					weapon.Energy++
				}
			},
		},
		{
			Name: "Reinforce",
			Desc: "+20 max HP and heal 20",
			Apply: func(w *World) {
				w.Player.MaxHP += 20
				w.Player.HP = math.Min(w.Player.MaxHP, w.Player.HP+20)
			},
		},
		{
			Name: "Nitro",
			Desc: "+0.4 move speed",
			Apply: func(w *World) {
				w.Player.Speed += 0.4
			},
		},
		{
			Name: "Repair",
			Desc: "Heal 40 HP",
			Apply: func(w *World) {
				w.Player.HP = math.Min(w.Player.MaxHP, w.Player.HP+40)
			},
		},
	}
}
