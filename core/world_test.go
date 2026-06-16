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
	w := NewWorld(testSeed)

	if w.State != StatePlaying {
		t.Errorf("State = %v, want StatePlaying", w.State)
	}
	if w.Player == nil {
		t.Fatal("Player is nil")
	}
	if w.Player.HP != w.Player.MaxHP {
		t.Errorf("HP %.2f != MaxHP %.2f", w.Player.HP, w.Player.MaxHP)
	}
	if len(w.Player.Weapons) != 1 {
		t.Errorf("len(Weapons) = %d, want 1", len(w.Player.Weapons))
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
	w := NewWorld(testSeed)
	before := w.Player.Pos
	w.Update(noMove())
	if w.Player.Pos != before {
		t.Errorf("player moved when no input: %v -> %v", before, w.Player.Pos)
	}
}

func TestUpdate_DiagonalMoveIsNormalized(t *testing.T) {
	w := NewWorld(testSeed)
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
	w := NewWorld(testSeed)
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
	w := NewWorld(testSeed)
	for i := 1; i <= 5; i++ {
		w.Update(noMove())
		if w.Tick != i {
			t.Errorf("Tick = %d after %d updates, want %d", w.Tick, i, i)
		}
	}
}

// ── 3. Projectile firing + enemy kill ────────────────────────────────────────

func TestUpdate_WeaponFiresAndKillsEnemy(t *testing.T) {
	w := NewWorld(testSeed)

	// Place weapon at energy=0: stats give FireInterval=45, Damage=5, Range=220.
	weapon := w.Player.Weapons[0]
	weapon.Energy = 0
	weapon.cooldown = 0

	stats := weapon.StatsFromEnergy()

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
	// Give plenty of room; weapon cooldown also resets every FireInterval ticks.
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
	// A gem should have been created (then immediately collected if player is
	// close — but the gem starts at the enemy position which is 50px away and
	// the player is at origin; pickup range is 28, so it won't be instantly
	// collected). Either way a gem was created; if it was consumed XP > 0.
	gemCreatedOrConsumed := len(w.Gems) > 0 || w.Player.XP > 0
	if !gemCreatedOrConsumed {
		t.Errorf("no gem created and no XP gained after kill")
	}
}

func TestUpdate_ProjectileCreatedBeforeHit(t *testing.T) {
	w := NewWorld(testSeed)
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

	// Run until a projectile appears.
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
	w := NewWorld(testSeed)

	// Place enemy on top of player; speed=0 so it doesn't chase.
	enemy := &Enemy{
		Pos:    w.Player.Pos, // exactly on the player
		HP:     1e9,          // invincible
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
	w := NewWorld(testSeed)
	enemy := &Enemy{
		Pos:    w.Player.Pos,
		HP:     1e9,
		Speed:  0,
		Radius: 12,
		Damage: 8,
		alive:  true,
	}
	w.Enemies = append(w.Enemies, enemy)

	// First tick: takes damage and gains invuln.
	w.Update(noMove())
	hpAfterFirst := w.Player.HP

	// While invuln > 0 the player should not take more damage.
	w.Update(noMove())
	if w.Player.HP != hpAfterFirst {
		t.Errorf("HP changed during invuln frames: %.2f -> %.2f", hpAfterFirst, w.Player.HP)
	}
}

func TestUpdate_GameOverWhenHPReachesZero(t *testing.T) {
	w := NewWorld(testSeed)

	// Place an enemy that deals enough damage per hit to kill the player in
	// one shot (HP=100, so Damage>=100).
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
	w := NewWorld(testSeed)
	enemy := &Enemy{
		Pos:    w.Player.Pos,
		HP:     1e9,
		Speed:  0,
		Radius: 12,
		Damage: 200,
		alive:  true,
	}
	w.Enemies = append(w.Enemies, enemy)

	// Kill the player.
	w.Update(noMove())
	if w.State != StateGameOver {
		t.Fatal("State is not GameOver after setup")
	}

	tickBefore := w.Tick

	// Further updates should be no-ops.
	w.Update(noMove())
	w.Update(noMove())

	if w.Tick != tickBefore {
		t.Errorf("Tick advanced after GameOver: was %d, now %d", tickBefore, w.Tick)
	}
}

// ── 5. XP pickup + level up ───────────────────────────────────────────────────

func TestUpdate_GemPickupGrantsXP(t *testing.T) {
	w := NewWorld(testSeed)

	// Place gem within pickup range (28px) of the player at origin.
	gem := &Gem{
		Pos:   geom.PointF{X: 10, Y: 0},
		Value: 5,
		alive: true,
	}
	w.Gems = append(w.Gems, gem)

	w.Update(noMove())

	// Gem should be consumed.
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
	w := NewWorld(testSeed)

	// XPToNext starts at 10. One gem with value >= 10 triggers a level-up.
	startLevel := w.Player.Level
	startXPToNext := w.Player.XPToNext

	gem := &Gem{
		Pos:   geom.PointF{X: 0, Y: 0}, // exactly at player
		Value: w.Player.XPToNext,        // exactly enough to level up
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
	if len(w.Choices) != 3 {
		t.Errorf("len(Choices) = %d, want 3", len(w.Choices))
	}

	// The world is paused: Update must be a no-op until a choice is made.
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
	w := NewWorld(testSeed)

	// A gem worth two thresholds (10 + ceil(10*1.25)=13 = 23) earns two levels.
	gem := &Gem{
		Pos:   geom.PointF{X: 0, Y: 0},
		Value: 23,
		alive: true,
	}
	w.Gems = append(w.Gems, gem)

	w.Update(noMove())

	if w.Player.Level != 3 { // 1 -> 3
		t.Fatalf("Level = %d, want 3 after two level-ups", w.Player.Level)
	}
	if w.State != StateLevelUp {
		t.Fatalf("State = %d, want StateLevelUp", w.State)
	}

	// First choice still leaves one pending: stays in StateLevelUp with new choices.
	w.ChooseUpgrade(0)
	if w.State != StateLevelUp {
		t.Errorf("State = %d after first of two choices, want StateLevelUp", w.State)
	}
	if len(w.Choices) != 3 {
		t.Errorf("len(Choices) = %d after re-roll, want 3", len(w.Choices))
	}

	// Second choice resumes play.
	w.ChooseUpgrade(0)
	if w.State != StatePlaying {
		t.Errorf("State = %d after second choice, want StatePlaying", w.State)
	}
}

func TestUpgradeCatalog_HealsAreCappedAtMaxHP(t *testing.T) {
	for _, u := range upgradeCatalog() {
		w := NewWorld(testSeed)
		w.Player.HP = w.Player.MaxHP // already full
		u.Apply(w)
		if w.Player.HP > w.Player.MaxHP {
			t.Errorf("upgrade %q raised HP %.2f above MaxHP %.2f", u.Name, w.Player.HP, w.Player.MaxHP)
		}
	}
}

// ── 6. Determinism ────────────────────────────────────────────────────────────

func TestDeterminism_SameSeedSameSpawnPositions(t *testing.T) {
	const ticks = 200

	worldA := NewWorld(testSeed)
	worldB := NewWorld(testSeed)

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

	worldA := NewWorld(testSeed)
	worldB := NewWorld(testSeed + 1)

	for i := 0; i < ticks; i++ {
		worldA.Update(noMove())
		worldB.Update(noMove())
	}

	// It's astronomically unlikely all enemies spawn in the exact same place.
	allSame := true
	n := len(worldA.Enemies)
	if n == 0 || n != len(worldB.Enemies) {
		// Different enemy counts are already evidence of divergence.
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

// ── 7. Enemy chases player ────────────────────────────────────────────────────

func TestUpdate_EnemyChasesPlayer(t *testing.T) {
	w := NewWorld(testSeed)

	// Put enemy to the right of the player.
	startEnemyPos := geom.PointF{X: 100, Y: 0}
	enemy := &Enemy{
		Pos:     startEnemyPos,
		HP:      1e9,
		Speed:   1.2,
		Radius:  12,
		Damage:  0, // no damage so we can run freely
		XPValue: 0,
		alive:   true,
	}
	w.Enemies = append(w.Enemies, enemy)

	w.Update(noMove())

	// Enemy should have moved toward player (origin), so X should decrease.
	if enemy.Pos.X >= startEnemyPos.X {
		t.Errorf("enemy did not move toward player: X %.2f -> %.2f", startEnemyPos.X, enemy.Pos.X)
	}
}

// ── 8. Projectile life expiry ─────────────────────────────────────────────────

func TestUpdate_ProjectileExpiresAfterLifeTicks(t *testing.T) {
	w := NewWorld(testSeed)

	// Inject a projectile with Life=1 that won't hit anything (aimed away).
	p := &Projectile{
		Pos:    geom.PointF{X: 0, Y: 0},
		Vel:    geom.PointF{X: 0, Y: -6}, // moving away from origin
		Damage: 5,
		Radius: 5,
		Life:   1,
		alive:  true,
	}
	w.Projectiles = append(w.Projectiles, p)

	w.Update(noMove()) // Life goes to 0, projectile should die and be compacted.

	if len(w.Projectiles) != 0 {
		t.Errorf("expired projectile still in slice: %d remaining", len(w.Projectiles))
	}
}
