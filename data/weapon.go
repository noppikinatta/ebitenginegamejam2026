package data

import "github.com/noppikinatta/ebitenginegamejam2026/core"

// weaponParams returns the balance preset for every weapon kind. Edit these to
// tune combat feel without touching game logic. See core.WeaponParams for the
// formulae these numbers feed.
func weaponParams() map[core.WeaponKind]core.WeaponParams {
	return map[core.WeaponKind]core.WeaponParams{
		core.KindCannon: {
			BaseDamage:   20,
			BaseInterval: 720, MinInterval: 6, // 12s between shots at fireMult=1
			ProjSpeed: 6, ProjMaxDist: 600, ProjRadius: 6, BaseRange: 360,
			Sprite: core.SpriteCannon, ProjDrawW: 8, ProjDrawH: 14, ProjFaceVelocity: true, // chunky shell, points along travel
			LevelMult: 1.2,
		},
		core.KindShotgun: {
			BaseDamage:   8,                   // ×4 pellets per shot
			BaseInterval: 720, MinInterval: 8, // 12s
			ProjSpeed: 5, ProjMaxDist: 400, ProjRadius: 2, BaseRange: 250,
			Pellets: 4, SpreadRad: 0.3, // simultaneous fixed 4-pellet spread
			Sprite: core.SpriteShotgun, ProjDrawW: 6, ProjDrawH: 6, // small round pellet
			LevelMult: 1.2,
		},
		core.KindSniper: {
			BaseDamage:   10,
			BaseInterval: 960, MinInterval: 20, // 16s
			ProjSpeed: 10, ProjMaxDist: 800, ProjRadius: 2, BaseRange: 600,
			Target: core.TargetFarthest,                                                    // picks off the farthest enemy in range
			Sprite: core.SpriteSniper, ProjDrawW: 4, ProjDrawH: 16, ProjFaceVelocity: true, // long thin dart
			LevelMult: 1.2,
		},
		core.KindLaser: {
			BaseDamage:   1,                     // per tick; ×BeamBaseDuration(30) = 30 over a burst
			BaseInterval: 1440, MinInterval: 15, // 24s
			BaseRange:        400,
			Target:           core.TargetFarthest, // sweeps onto the farthest enemy in range
			BeamBaseLength:   400,
			BeamBaseWidth:    6,
			BeamBaseDuration: 30,
			LevelMult:        1.2,
		},
		core.KindGatling: {
			BaseDamage:   2,
			BaseInterval: 480, MinInterval: 6, // 8s between bursts
			ProjSpeed: 5, ProjMaxDist: 500, ProjRadius: 2,
			Pellets: 10, SpreadRad: 0.2, SpreadRandom: true, BurstGap: 3, // staggered random stream
			Aim:    core.AimForward,                                // never locks on; always fires forward
			Sprite: core.SpriteGatling, ProjDrawW: 6, ProjDrawH: 6, // small round slug
			LevelMult: 1.2,
		},
		core.KindGrenade: {
			BaseDamage:   0,                     // no contact damage; all damage is the explosion
			BaseInterval: 1800, MinInterval: 30, // 30s
			ProjSpeed: 2, ProjMaxDist: 300, // lobbed slowly; expires after 60 ticks
			Aim:           core.AimOutward,                          // never locks on; always fires outward
			ExplodeRadius: 64, ExplodeDamage: 15, PassThrough: true, // flies through; detonates where it lands
			Sprite: core.SpriteGrenade, ProjDrawW: 14, ProjDrawH: 14, // fat round shell
			LevelMult: 1.2,
		},
		core.KindCIWS: {
			BaseDamage:   2,
			BaseInterval: 480, MinInterval: 6, // 8s between bursts
			ProjSpeed: 8, ProjMaxDist: 200, ProjRadius: 2, BaseRange: 200, // very short range point defence
			Pellets: 10, SpreadRad: 0.1, SpreadRandom: true, BurstGap: 2, // tight rapid burst
			HoldWhenNoTarget: true,                                        // stays charged until something enters range
			Sprite:           core.SpriteCIWS, ProjDrawW: 6, ProjDrawH: 6, // small round tracer
			LevelMult: 1.2,
		},
		core.KindMissile: {
			BaseDamage:   8,                    // contact damage
			BaseInterval: 960, MinInterval: 20, // 16s
			ProjSpeed: 4, ProjMaxDist: 600, ProjRadius: 6, BaseRange: 400, // slow shell, long lock range
			ExplodeRadius: 48, ExplodeDamage: 10, // smaller blast than the grenade, only if it expires unhit
			Mover:  core.NewHomingMover(0.3, 6, 15),                                         // flies straight 15 ticks (boost-out), then homes (turn 0.3, cruise 6)
			Sprite: core.SpriteMissile, ProjDrawW: 8, ProjDrawH: 12, ProjFaceVelocity: true, // larger shell, turns to face its target
			LevelMult: 1.2,
		},
	}
}

// powerCurve maps the connected consumer tile count to a fire-rate multiplier.
// Fewer tiles concentrate power into faster fire; more tiles dilute it. Points
// must be sorted ascending by Tiles; outside the range the end values clamp.
// These are placeholder breakpoints — tune them during playtesting.
func powerCurve() []core.PowerPoint {
	return []core.PowerPoint{
		{Tiles: 10, Mult: 4.0},
		{Tiles: 32, Mult: 1.0},
		{Tiles: 40, Mult: 0.5},
	}
}
