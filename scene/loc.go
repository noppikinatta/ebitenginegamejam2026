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

// Flavour names originate in core as English strings; resolve them through the
// CSVs by slug, falling back to the original literal if no entry exists.
func doctorNameL(name string) string { return lang.TextWithDefault("doctor-"+slug(name), name) }
func junkNameL(name string) string   { return lang.TextWithDefault("junk-"+slug(name), name) }
func bossNameL(name string) string   { return lang.TextWithDefault("boss-"+slug(name), name) }

// junkDescL returns the per-device description for a junk, keyed by
// junk-<slug>-desc. If a device has no specific entry yet it falls back to the
// generic junk-desc / junk-desc-tall copy, so new junk still shows something.
func junkDescL(name string, tall bool) string {
	generic := "junk-desc"
	if tall {
		generic = "junk-desc-tall"
	}
	return lang.TextWithDefault("junk-"+slug(name)+"-desc", lang.Text(generic))
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
	case core.OfferNippers:
		return lang.ExecuteTemplate("offer-nippers", map[string]any{"N": it.Amount})
	default: // OfferAddJunk
		return junkNameL(it.Text)
	}
}
