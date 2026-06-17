package core

import (
	"math"
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
)

// ── helpers ──────────────────────────────────────────────────────────────────

const testSeed int64 = 42

func noMove() geom.PointF { return geom.PointF{} }

// almostEqual returns true when |a-b| < eps.
func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

// ── 1. NewWorld initial state ─────────────────────────────────────────────────

func TestNewWorld_InitialState(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	if w.State != StatePlaying {
		t.Errorf("State = %v, want StatePlaying", w.State)
	}
	if w.Player == nil {
		t.Fatal("Player is nil")
	}
	if w.Player.HP != w.Player.MaxHP {
		t.Errorf("HP %.2f != MaxHP %.2f", w.Player.HP, w.Player.MaxHP)
	}
	// Generated turret should have at least one weapon.
	if len(w.Player.Weapons) < 1 {
		t.Errorf("len(Weapons) = %d, want >= 1", len(w.Player.Weapons))
	}
	if w.Player.Level != 1 {
		t.Errorf("Level = %d, want 1", w.Player.Level)
	}
	if w.Tick != 0 {
		t.Errorf("Tick = %d, want 0", w.Tick)
	}
	if w.Kills != 0 {
		t.Errorf("Kills = %d, want 0", w.Kills)
	}
}

// ── 2. Movement ───────────────────────────────────────────────────────────────

func TestUpdate_ZeroMoveDoesNotMovePlayer(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	before := w.Player.Pos
	w.Update(noMove())
	if w.Player.Pos != before {
		t.Errorf("player moved when no input: %v -> %v", before, w.Player.Pos)
	}
}

func TestUpdate_DiagonalMoveIsNormalized(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	startPos := w.Player.Pos
	speed := w.Player.Speed

	// diagonal (1,1) has magnitude sqrt(2); after normalisation each axis is
	// 1/sqrt(2), so the total displacement must equal exactly Speed.
	w.Update(geom.PointF{X: 1, Y: 1})

	dx := w.Player.Pos.X - startPos.X
	dy := w.Player.Pos.Y - startPos.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if !almostEqual(dist, speed, 1e-9) {
		t.Errorf("diagonal move distance = %.6f, want %.6f (Speed)", dist, speed)
	}
}

func TestUpdate_AxisAlignedMoveIsExactlySpeed(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	startPos := w.Player.Pos
	speed := w.Player.Speed

	w.Update(geom.PointF{X: 1, Y: 0})

	dx := w.Player.Pos.X - startPos.X
	if !almostEqual(dx, speed, 1e-9) {
		t.Errorf("horizontal move dx = %.6f, want %.6f (Speed)", dx, speed)
	}
	if w.Player.Pos.Y != startPos.Y {
		t.Errorf("Y changed unexpectedly: %.6f -> %.6f", startPos.Y, w.Player.Pos.Y)
	}
}

func TestUpdate_TickIncrements(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	for i := 1; i <= 5; i++ {
		w.Update(noMove())
		if w.Tick != i {
			t.Errorf("Tick = %d after %d updates, want %d", w.Tick, i, i)
		}
	}
}

// ── 3. Projectile firing + enemy kill ────────────────────────────────────────

func TestUpdate_WeaponFiresAndKillsEnemy(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	// Place weapon at energy=0: stats give FireInterval=45, Damage=5, Range=220.
	weapon := w.Player.Weapons[0]
	weapon.Energy = 0
	weapon.cooldown = 0

	stats := weapon.StatsFromEnergy(w.cfg.Weapons[weapon.Kind])

	// Place a weak enemy just within range.
	enemy := &Enemy{
		Pos:     geom.PointF{X: 50, Y: 0},
		HP:      4, // will die on first hit (Damage=5)
		Speed:   0, // stationary so it doesn't wander
		Radius:  12,
		Damage:  0,
		XPValue: 3,
		alive:   true,
	}
	w.Enemies = append(w.Enemies, enemy)

	// Sanity-check the enemy is in range.
	dist := w.Player.Pos.Distance(enemy.Pos)
	if dist > stats.Range {
		t.Fatalf("enemy distance %.2f > range %.2f; adjust test setup", dist, stats.Range)
	}

	// projectile speed is 6 px/tick, enemy is 50px away.
	// It should reach the enemy (radius 12) within ceil((50-12)/6)+1 ≈ 8 ticks.
	const maxTicks = 200
	killed := false
	for i := 0; i < maxTicks; i++ {
		w.Update(noMove())
		if w.Kills > 0 {
			killed = true
			break
		}
	}

	if !killed {
		t.Fatalf("enemy was not killed after %d ticks", maxTicks)
	}
	if w.Kills != 1 {
		t.Errorf("Kills = %d, want 1", w.Kills)
	}
	// After compact(), no dead enemies remain in the slice.
	for _, e := range w.Enemies {
		if !e.alive {
			t.Errorf("dead enemy still in Enemies slice after compact")
		}
	}
	gemCreatedOrConsumed := len(w.Gems) > 0 || w.Player.XP > 0
	if !gemCreatedOrConsumed {
		t.Errorf("no gem created and no XP gained after kill")
	}
}

func TestUpdate_ProjectileCreatedBeforeHit(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	weapon := w.Player.Weapons[0]
	weapon.Energy = 0
	weapon.cooldown = 0

	// Place a far-away enemy so the projectile is in-flight for several ticks.
	enemy := &Enemy{
		Pos:     geom.PointF{X: 200, Y: 0},
		HP:      1000,
		Speed:   0,
		Radius:  12,
		Damage:  0,
		XPValue: 0,
		alive:   true,
	}
	w.Enemies = append(w.Enemies, enemy)

	const maxTicks = 200
	fired := false
	for i := 0; i < maxTicks; i++ {
		w.Update(noMove())
		if len(w.Projectiles) > 0 || len(w.Enemies) == 0 {
			fired = true
			break
		}
	}
	if !fired {
		t.Fatalf("no projectile was fired in %d ticks", maxTicks)
	}
}

// ── 4. Contact damage + game over ────────────────────────────────────────────

func TestUpdate_ContactDamageAndInvuln(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	enemy := &Enemy{
		Pos:    w.Player.Pos, // exactly on the player
		HP:     1e9,
		Speed:  0,
		Radius: 12,
		Damage: 8,
		alive:  true,
	}
	w.Enemies = append(w.Enemies, enemy)
	startHP := w.Player.HP

	w.Update(noMove())

	if w.Player.HP >= startHP {
		t.Errorf("HP did not decrease after contact: %.2f", w.Player.HP)
	}
	if w.Player.invuln == 0 {
		t.Errorf("invuln not set after taking damage")
	}
}

func TestUpdate_InvulnPreventsDoubleHit(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	enemy := &Enemy{
		Pos:    w.Player.Pos,
		HP:     1e9,
		Speed:  0,
		Radius: 12,
		Damage: 8,
		alive:  true,
	}
	w.Enemies = append(w.Enemies, enemy)

	w.Update(noMove())
	hpAfterFirst := w.Player.HP

	w.Update(noMove())
	if w.Player.HP != hpAfterFirst {
		t.Errorf("HP changed during invuln frames: %.2f -> %.2f", hpAfterFirst, w.Player.HP)
	}
}

func TestUpdate_GameOverWhenHPReachesZero(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	enemy := &Enemy{
		Pos:    w.Player.Pos,
		HP:     1e9,
		Speed:  0,
		Radius: 12,
		Damage: 200, // overkill
		alive:  true,
	}
	w.Enemies = append(w.Enemies, enemy)

	w.Update(noMove())

	if w.State != StateGameOver {
		t.Errorf("State = %v after lethal hit, want StateGameOver", w.State)
	}
	if w.Player.HP != 0 {
		t.Errorf("HP = %.2f after death, want 0", w.Player.HP)
	}
}

func TestUpdate_IsNoOpAfterGameOver(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	enemy := &Enemy{
		Pos:    w.Player.Pos,
		HP:     1e9,
		Speed:  0,
		Radius: 12,
		Damage: 200,
		alive:  true,
	}
	w.Enemies = append(w.Enemies, enemy)

	w.Update(noMove())
	if w.State != StateGameOver {
		t.Fatal("State is not GameOver after setup")
	}

	tickBefore := w.Tick
	w.Update(noMove())
	w.Update(noMove())

	if w.Tick != tickBefore {
		t.Errorf("Tick advanced after GameOver: was %d, now %d", tickBefore, w.Tick)
	}
}

// ── 5. XP pickup + level up ───────────────────────────────────────────────────

func TestUpdate_GemPickupGrantsXP(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	gem := &Gem{
		Pos:   geom.PointF{X: 10, Y: 0},
		Value: 5,
		alive: true,
	}
	w.Gems = append(w.Gems, gem)

	w.Update(noMove())

	for _, g := range w.Gems {
		if g == gem && g.alive {
			t.Errorf("gem still alive after being within pickup range")
		}
	}
	if w.Player.XP < 5 {
		t.Errorf("XP = %.2f after pickup, want >= 5", w.Player.XP)
	}
}

func TestUpdate_LevelUpPausesForChoice(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	startLevel := w.Player.Level
	startXPToNext := w.Player.XPToNext

	gem := &Gem{
		Pos:   geom.PointF{X: 0, Y: 0},
		Value: w.Player.XPToNext,
		alive: true,
	}
	w.Gems = append(w.Gems, gem)

	w.Update(noMove())

	if w.Player.Level != startLevel+1 {
		t.Errorf("Level = %d after level-up, want %d", w.Player.Level, startLevel+1)
	}
	if w.Player.XPToNext <= startXPToNext {
		t.Errorf("XPToNext did not grow: was %.2f, now %.2f", startXPToNext, w.Player.XPToNext)
	}
	if w.State != StateLevelUp {
		t.Errorf("State = %d after level-up, want StateLevelUp (%d)", w.State, StateLevelUp)
	}
	if len(w.Choices) == 0 {
		t.Errorf("no purge choices after level-up")
	}

	// World is paused: Update must be a no-op until a choice is made.
	tickBefore := w.Tick
	w.Update(geom.PointF{X: 1})
	if w.Tick != tickBefore {
		t.Errorf("Tick advanced from %d to %d while awaiting a choice", tickBefore, w.Tick)
	}

	// Choosing resumes play.
	w.ChooseUpgrade(0)
	if w.State != StatePlaying {
		t.Errorf("State = %d after choosing, want StatePlaying (%d)", w.State, StatePlaying)
	}
	if len(w.Choices) != 0 {
		t.Errorf("Choices not cleared after choosing: %d remain", len(w.Choices))
	}
}

func TestChooseUpgrade_MultipleLevelUpsQueueChoices(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	// A gem worth two thresholds (10 + ceil(10*1.25)=13 = 23) earns two levels.
	gem := &Gem{
		Pos:   geom.PointF{X: 0, Y: 0},
		Value: 23,
		alive: true,
	}
	w.Gems = append(w.Gems, gem)

	w.Update(noMove())

	if w.Player.Level != 3 {
		t.Fatalf("Level = %d, want 3 after two level-ups", w.Player.Level)
	}
	if w.State != StateLevelUp {
		t.Fatalf("State = %d, want StateLevelUp", w.State)
	}

	// First choice still leaves one pending.
	w.ChooseUpgrade(0)
	if w.State != StateLevelUp {
		t.Errorf("State = %d after first of two choices, want StateLevelUp", w.State)
	}

	// Second choice resumes play.
	w.ChooseUpgrade(0)
	if w.State != StatePlaying {
		t.Errorf("State = %d after second choice, want StatePlaying", w.State)
	}
}

// ── 6. Level-up: doctors add tiles (or hand out nippers) ─────────────────────

func TestRollChoices_ProducesThreeOffers(t *testing.T) {
	w := NewWorld(testSeed, testConfig())
	w.rollChoices()
	if len(w.Choices) != 3 {
		t.Errorf("rollChoices produced %d offers, want 3", len(w.Choices))
	}
}

func TestDoctorChoices_HaveValidOutcome(t *testing.T) {
	// Every offer must do one of: grow the turret (tile bundle), give nippers,
	// or upgrade at least one existing weapon's Level.
	w := NewWorld(testSeed, testConfig())
	w.rollChoices()

	for i, c := range w.Choices {
		beforeTiles := w.turret.TileCount()
		beforeNippers := w.Player.Nippers
		beforeLevel := totalWeaponLevel(w)
		c.Apply(w)
		grewTurret := w.turret.TileCount() > beforeTiles
		gaveNippers := w.Player.Nippers > beforeNippers
		upgradedWeapon := totalWeaponLevel(w) > beforeLevel
		if !grewTurret && !gaveNippers && !upgradedWeapon {
			t.Errorf("choice %d (%q) had no effect: tiles=%d nippers=%d levels=%d",
				i, c.Name, w.turret.TileCount()-beforeTiles,
				w.Player.Nippers-beforeNippers, totalWeaponLevel(w)-beforeLevel)
		}
	}
}

// totalWeaponLevel sums the Level field of all active weapons in the turret.
func totalWeaponLevel(w *World) int {
	total := 0
	for _, wp := range w.turret.ActiveWeapons() {
		total += wp.Level
	}
	return total
}

func TestRollChoices_AtCapNeverAddsTiles(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	// Grow the turret past the cap with junk tiles.
	for w.turret.TileCount() < w.cfg.MaxTurretTiles {
		if _, ok := w.turret.AddTile(Junk{DeviceName: "Toaster"}, w.rng); !ok {
			t.Fatal("AddTile ran out of room before reaching the cap")
		}
	}

	w.rollChoices()
	// At the cap no offer may grow the turret; each gives nippers or upgrades weapons.
	for i, c := range w.Choices {
		beforeTiles := w.turret.TileCount()
		c.Apply(w)
		if w.turret.TileCount() > beforeTiles {
			t.Errorf("offer %d (%q) grew the turret past the cap", i, c.Name)
		}
	}
}

// ── 7. Determinism ────────────────────────────────────────────────────────────

func TestDeterminism_SameSeedSameSpawnPositions(t *testing.T) {
	const ticks = 200

	worldA := NewWorld(testSeed, testConfig())
	worldB := NewWorld(testSeed, testConfig())

	for i := 0; i < ticks; i++ {
		worldA.Update(noMove())
		worldB.Update(noMove())
	}

	if len(worldA.Enemies) != len(worldB.Enemies) {
		t.Fatalf("enemy count differs: %d vs %d", len(worldA.Enemies), len(worldB.Enemies))
	}
	for i := range worldA.Enemies {
		ea := worldA.Enemies[i]
		eb := worldB.Enemies[i]
		if ea.Pos.X != eb.Pos.X || ea.Pos.Y != eb.Pos.Y {
			t.Errorf("enemy[%d] pos differs: %v vs %v", i, ea.Pos, eb.Pos)
		}
	}
}

func TestDeterminism_DifferentSeedsDifferentSpawns(t *testing.T) {
	const ticks = 200

	worldA := NewWorld(testSeed, testConfig())
	worldB := NewWorld(testSeed+1, testConfig())

	for i := 0; i < ticks; i++ {
		worldA.Update(noMove())
		worldB.Update(noMove())
	}

	allSame := true
	n := len(worldA.Enemies)
	if n == 0 || n != len(worldB.Enemies) {
		allSame = false
	} else {
		for i := range worldA.Enemies {
			if worldA.Enemies[i].Pos.X != worldB.Enemies[i].Pos.X {
				allSame = false
				break
			}
		}
	}
	if allSame {
		t.Errorf("different seeds produced identical enemy positions — RNG may be broken")
	}
}

// ── 8. FacingAngle tracks movement ───────────────────────────────────────────

func TestUpdate_FacingAngleTracksMovement(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	// Move right: angle should be 0.
	w.Update(geom.PointF{X: 1, Y: 0})
	if !almostEqual(w.Player.FacingAngle, 0, 1e-9) {
		t.Errorf("FacingAngle after rightward move = %.6f, want ~0", w.Player.FacingAngle)
	}

	// Zero move: FacingAngle must not change.
	w.Update(geom.PointF{X: 0, Y: 0})
	if !almostEqual(w.Player.FacingAngle, 0, 1e-9) {
		t.Errorf("FacingAngle changed on zero move: %.6f, want ~0", w.Player.FacingAngle)
	}

	// Move up (Y negative in screen coords): angle should be -pi/2.
	w.Update(geom.PointF{X: 0, Y: -1})
	if !almostEqual(w.Player.FacingAngle, -math.Pi/2, 1e-9) {
		t.Errorf("FacingAngle after upward move = %.6f, want ~-pi/2 (%.6f)", w.Player.FacingAngle, -math.Pi/2)
	}
}

// ── 9. Enemy chases player ───────────────────────────────────────────────────

func TestUpdate_EnemyChasesPlayer(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	startEnemyPos := geom.PointF{X: 100, Y: 0}
	enemy := &Enemy{
		Pos:     startEnemyPos,
		HP:      1e9,
		Speed:   1.2,
		Radius:  12,
		Damage:  0,
		XPValue: 0,
		alive:   true,
	}
	w.Enemies = append(w.Enemies, enemy)

	w.Update(noMove())

	if enemy.Pos.X >= startEnemyPos.X {
		t.Errorf("enemy did not move toward player: X %.2f -> %.2f", startEnemyPos.X, enemy.Pos.X)
	}
}

// ── 10. Projectile life expiry ────────────────────────────────────────────────

func TestUpdate_ProjectileExpiresAfterLifeTicks(t *testing.T) {
	w := NewWorld(testSeed, testConfig())

	p := &Projectile{
		Pos:    geom.PointF{X: 0, Y: 0},
		Vel:    geom.PointF{X: 0, Y: -6},
		Damage: 5,
		Radius: 5,
		Life:   1,
		alive:  true,
	}
	w.Projectiles = append(w.Projectiles, p)

	w.Update(noMove())

	if len(w.Projectiles) != 0 {
		t.Errorf("expired projectile still in slice: %d remaining", len(w.Projectiles))
	}
}
