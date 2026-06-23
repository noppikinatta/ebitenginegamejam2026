package scene

import (
	"strings"

	"github.com/noppikinatta/ebitenginegamejam2026/core"
	"github.com/noppikinatta/ebitenginegamejam2026/lang"
)

// slug converts a display name into a CSV key fragment: lower-cased with spaces
// turned into hyphens (e.g. "Five-storied Pagoda" -> "five-storied-pagoda").
func slug(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

// kindSlug is the key fragment for a weapon kind (e.g. "cannon").
func kindSlug(k core.WeaponKind) string { return strings.ToLower(k.String()) }

// weaponName / weaponDescL return the localised name and description for a
// weapon kind, keyed by weapon-<kind>-name / -desc.
func weaponName(k core.WeaponKind) string { return lang.Text("weapon-" + kindSlug(k) + "-name") }
func weaponDescL(k core.WeaponKind) string {
	return lang.Text("weapon-" + kindSlug(k) + "-desc")
}

// doctorNameL formats a doctor's full name from its title (e.g. "Doctor") and
// random letter via a per-language template (doctor-name-<title>), so the letter
// can sit before or after the title depending on the language.
func doctorNameL(title, alphabet string) string {
	return lang.ExecuteTemplate("doctor-name-"+slug(title), map[string]any{"Alphabet": alphabet})
}

// Flavour names originate in core as English strings; resolve them through the
// CSVs by slug, falling back to the original literal if no entry exists.
func junkNameL(name string) string { return lang.TextWithDefault("junk-"+slug(name), name) }
func bossNameL(name string) string { return lang.TextWithDefault("boss-"+slug(name), name) }

// junkDescL returns the per-device description for a junk, keyed by
// junk-<slug>-desc. Every junk device has its own entry; a device without one
// degrades to no description rather than a NO_TMPL marker.
func junkDescL(name string) string {
	return lang.TextWithDefault("junk-"+slug(name)+"-desc", "")
}

// offerItemText returns the localised display name for one proposal line,
// derived from the item's structured fields rather than its baked-in English
// Text.
func offerItemText(it core.OfferItem) string {
	switch it.Kind {
	case core.OfferAddWeapon, core.OfferUpgrade:
		return weaponName(it.Weapon)
	case core.OfferAddCapacitor:
		return lang.Text("comp-capacitor")
	case core.OfferAddRepairUnit:
		return lang.Text("comp-repair-unit")
	case core.OfferAddArmor:
		return lang.Text("comp-armor")
	case core.OfferNippers:
		return lang.ExecuteTemplate("offer-nippers", map[string]any{"N": it.Amount})
	default: // OfferAddJunk
		return junkNameL(it.Text)
	}
}

// offerItemDesc returns the localised one-line description for a proposal line,
// matching the copy shown for the same component in the pause/cut info panel
// (see pauseTileInfo). Nippers carry their own key since they are not a tile.
func offerItemDesc(it core.OfferItem) string {
	switch it.Kind {
	case core.OfferAddWeapon, core.OfferUpgrade:
		return weaponDescL(it.Weapon)
	case core.OfferAddCapacitor:
		return lang.Text("comp-capacitor-desc")
	case core.OfferAddRepairUnit:
		return lang.Text("comp-repair-unit-desc")
	case core.OfferAddArmor:
		return lang.Text("comp-armor-desc")
	case core.OfferNippers:
		return lang.Text("offer-nippers-desc")
	default: // OfferAddJunk
		return junkDescL(it.Text)
	}
}
