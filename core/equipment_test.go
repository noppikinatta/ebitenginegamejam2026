package core

import (
	"testing"

	"github.com/noppikinatta/ebitenginegamejam2026/geom"
	"github.com/noppikinatta/ebitenginegamejam2026/hexmap"
)

func TestModifierAdd_SumsAllFields(t *testing.T) {
	a := Modifier{DamageBonus: 0.5, HPRegen: 1, Armor: 2}
	b := Modifier{DamageBonus: 0.25, HPRegen: 3, Armor: 4}
	got := a.Add(b)
	want := Modifier{DamageBonus: 0.75, HPRegen: 4, Armor: 6}
	if got != want {
		t.Errorf("Add = %+v, want %+v", got, want)
	}
}

func TestEquipmentMods(t *testing.T) {
	if m := (RepairUnit{HealAmount: 1}).Mods(); m.HPRegen != 1 {
		t.Errorf("RepairUnit HPRegen = %v, want 1", m.HPRegen)
	}
	if m := (Armor{Reduction: 1}).Mods(); m.Armor != 1 {
		t.Errorf("Armor Armor = %v, want 1", m.Armor)
	}
}

// equipTurret builds a turret with comp on a single tile connected to the
// generator, so Turret.Modifiers() reflects that one component.
func equipTurret(comp Component) *Turret {
	gen := hexmap.IdxXY(0, 0)
	adj := hexmap.IdxXY(1, 0)
	tiles := map[hexmap.Index]*Tile{
		gen: makeTile(Wire{}),
		adj: makeTile(comp),
	}
	return NewTurret(tiles, []hexmap.Index{gen}, 100)
}

func TestRepairPlayer_HealsOnInterval(t *testing.T) {
	cfg := testConfig()
	cfg.RepairInterval = 5
	w := &World{Player: &Player{HP: 50, MaxHP: 100}, cfg: cfg, turret: equipTurret(RepairUnit{HealAmount: 1})}

	for i := 0; i < 4; i++ {
		w.repairPlayer()
	}
	if w.Player.HP != 50 {
		t.Errorf("HP healed early: got %v, want 50 before the interval elapses", w.Player.HP)
	}
	w.repairPlayer() // 5th tick: heal cycle fires
	if w.Player.HP != 51 {
		t.Errorf("HP = %v, want 51 after one repair cycle", w.Player.HP)
	}
}

func TestRepairPlayer_CapsAtMaxHP(t *testing.T) {
	cfg := testConfig()
	cfg.RepairInterval = 1
	w := &World{Player: &Player{HP: 100, MaxHP: 100}, cfg: cfg, turret: equipTurret(RepairUnit{HealAmount: 5})}
	w.repairPlayer()
	if w.Player.HP != 100 {
		t.Errorf("HP = %v, want 100 (capped at MaxHP)", w.Player.HP)
	}
}

func TestRepairPlayer_NoUnitsNoHeal(t *testing.T) {
	cfg := testConfig()
	cfg.RepairInterval = 1
	w := &World{Player: &Player{HP: 50, MaxHP: 100}, cfg: cfg, turret: equipTurret(Wire{})}
	w.repairPlayer()
	if w.Player.HP != 50 {
		t.Errorf("HP = %v, want 50 with no repair units", w.Player.HP)
	}
}

func TestDamagePlayer_ArmorReducesDamage(t *testing.T) {
	w := &World{Player: &Player{HP: 100, MaxHP: 100, Pos: geom.PointF{}}, State: StatePlaying, turret: equipTurret(Armor{Reduction: 1})}
	w.damagePlayer(10)
	if w.Player.HP != 91 {
		t.Errorf("HP = %v, want 91 (10 damage - 1 armor)", w.Player.HP)
	}
}

func TestDamagePlayer_ArmorMinimumOneDamage(t *testing.T) {
	w := &World{Player: &Player{HP: 100, MaxHP: 100, Pos: geom.PointF{}}, State: StatePlaying, turret: equipTurret(Armor{Reduction: 5})}
	w.damagePlayer(3) // more armor than the hit
	if w.Player.HP != 99 {
		t.Errorf("HP = %v, want 99 (at least 1 damage lands)", w.Player.HP)
	}
}

func TestDamagePlayer_NoArmorUnchanged(t *testing.T) {
	w := &World{Player: &Player{HP: 100, MaxHP: 100, Pos: geom.PointF{}}, State: StatePlaying, turret: equipTurret(Wire{})}
	w.damagePlayer(10)
	if w.Player.HP != 90 {
		t.Errorf("HP = %v, want 90 with no armor", w.Player.HP)
	}
}
