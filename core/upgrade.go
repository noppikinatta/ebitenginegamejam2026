package core

// OfferKind classifies one line (item) of a doctor's level-up proposal, so the
// scene can draw an icon and a label for it without knowing core internals.
type OfferKind int

const (
	OfferAddWeapon    OfferKind = iota // add a new weapon tile
	OfferAddJunk                       // add a junk tile
	OfferAddCapacitor                  // add a capacitor tile
	OfferUpgrade                       // level up an existing weapon
	OfferNippers                       // grant spare nippers
)

// OfferItem is one line of a doctor's proposal. Weapon is meaningful for
// OfferAddWeapon and OfferUpgrade (it drives both the icon and the name); Amount
// carries the nipper count for OfferNippers. Text is a display-name fallback
// (weapon/junk name); the scene prefers to localise via the structured fields
// and only falls back to Text when no key is defined.
type OfferItem struct {
	Kind   OfferKind
	Weapon WeaponKind
	Amount int
	Text   string
}

// Upgrade is a single doctor's level-up proposal: a list of items (a mix of tile
// additions and weapon upgrades) that are all applied together when the proposal
// is chosen. Doctor is the flavour name shown as the card title.
//
// The struct stays display-agnostic: it carries the data the scene needs to
// render icons and labels, not Ebiten image keys (those live in the scene/asset
// layers). Apply performs every item's effect.
type Upgrade struct {
	Doctor string
	Items  []OfferItem
	Apply  func(*World)
}
