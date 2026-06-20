// Copyright 2022 noppikinatta
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asset

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/noppikinatta/ebitenginegamejam2026/sndpak"
)

// sePak is the obfuscated bundle of sound EFFECTS, keyed by base name (e.g.
// "fire"). The loose source files live in asset/sound/raw (gitignored) and are
// packed into this file by `make sound-pak`; only the pak is committed, so raw,
// directly-playable SE (often licensed free-asset files) are not exposed in the
// repository. See package sndpak — light obfuscation to respect asset licenses,
// not encryption.
//
//go:embed sound/se.pak
var sePak []byte

// BGM tracks. BGM is self-authored, so it has no license concern and is
// committed and embedded directly (no pak / obfuscation). There are two songs:
// the title track (opening + title screens) and the game track (in-game +
// result). Swap in real music by replacing the matching wav.
//
//go:embed sound/bgm_title.wav
var bgmTitleBytes []byte

//go:embed sound/bgm_game.wav
var bgmGameBytes []byte

const sampleRate int = 48000

var context *audio.Context

func init() {
	context = audio.NewContext(sampleRate)
	seBytes = map[Sound][]byte{}
}

// Sound identifies a one-shot sound effect; each Play spins up a fresh, cheap
// player so effects may overlap. Each weapon kind has its own fire SE.
type Sound int

const (
	SEFireCannon Sound = iota
	SEFireShotgun
	SEFireSniper
	SEFireLaser
	SEFireGatling
	SEFireGrenade
	SEFireCIWS
	SEFireMissile
	SEExplosion
	SEPlayerHit
)

// BGMTrack identifies a looping background track. PlayBGM switches between them.
type BGMTrack int

const (
	BGMTitle BGMTrack = iota // opening + title screens
	BGMGame                  // in-game + result screens
)

type fileType int

const (
	fileTypeWav fileType = iota
	fileTypeMp3
	fileTypeOgg
)

// soundSpec describes how to load one sound effect. pakName is the entry name
// inside sePak (the base name of the source file, e.g. "fire"); fileType selects
// the decoder for that entry's bytes.
type soundSpec struct {
	pakName  string
	sound    Sound
	fileType fileType
	volume   float64
}

// seSpecs are the one-shot effects, loaded from sePak. Each weapon kind has its
// own fire entry. BGM is handled separately (committed wav, not in the pak).
var seSpecs = []soundSpec{
	{"fire_cannon", SEFireCannon, fileTypeWav, 0.2},
	{"fire_shotgun", SEFireShotgun, fileTypeWav, 0.2},
	{"fire_sniper", SEFireSniper, fileTypeWav, 0.2},
	{"fire_laser", SEFireLaser, fileTypeWav, 0.2},
	{"fire_gatling", SEFireGatling, fileTypeWav, 0.15},
	{"fire_grenade", SEFireGrenade, fileTypeWav, 0.25},
	{"fire_ciws", SEFireCIWS, fileTypeWav, 0.15},
	{"fire_missile", SEFireMissile, fileTypeWav, 0.25},
	{"explosion", SEExplosion, fileTypeWav, 0.3},
	{"hit", SEPlayerHit, fileTypeWav, 0.4},
}

// bgmSpec describes a BGM track: its embedded bytes, format and volume.
type bgmSpec struct {
	track    BGMTrack
	resource []byte
	fileType fileType
	volume   float64
}

var bgmSpecs = []bgmSpec{
	{BGMTitle, bgmTitleBytes, fileTypeWav, 0.25},
	{BGMGame, bgmGameBytes, fileTypeWav, 0.25},
}

// seBytes holds the decoded PCM for each one-shot effect so PlaySound can create
// a fresh player per play (allowing the same SE to overlap). bgmPlayers holds one
// reusable looping player per track; currentBGM tracks which is playing so
// re-requesting the active track is a no-op (seamless across scenes that share a
// track).
var (
	seBytes    map[Sound][]byte
	seVolume   = map[Sound]float64{}
	bgmPlayers = map[BGMTrack]*audio.Player{}
	currentBGM = BGMTrack(-1)
	bgmPlaying bool
)

// LoadSounds decodes the BGM (committed wav) and the sound effects (unpacked
// from the embedded sePak). A failed track (missing pak entry or undecodable
// bytes) is logged and skipped so a single bad asset does not take down the
// whole game; SE can be swapped by repacking asset/sound/raw, BGM by replacing
// asset/sound/bgm.wav. If the pak is unreadable the SE are simply silent.
func LoadSounds() error {
	for _, b := range bgmSpecs {
		if err := loadBGM(b); err != nil {
			log.Printf("skip loading BGM %d: %v", b.track, err)
		}
	}

	blobs, err := sndpak.Unpack(sePak)
	if err != nil {
		log.Printf("cannot read SE pak (running without sound effects): %v", err)
		return nil
	}
	for _, s := range seSpecs {
		resource := blobs[s.pakName]
		if resource == nil {
			log.Printf("skip loading SE %q: not present in pak", s.pakName)
			continue
		}
		if err := loadSE(s, resource); err != nil {
			log.Printf("skip loading SE %q: %v", s.pakName, err)
			continue
		}
	}
	return nil
}

func decode(resource []byte, ftype fileType) (io.ReadSeeker, error) {
	switch ftype {
	case fileTypeWav:
		return wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(resource))
	case fileTypeMp3:
		return mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(resource))
	case fileTypeOgg:
		return vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(resource))
	default:
		return nil, errors.New("not supported filetype")
	}
}

// loadBGM decodes one background track into its reusable looping player.
func loadBGM(b bgmSpec) error {
	stream, err := decode(b.resource, b.fileType)
	if err != nil {
		return err
	}
	// One reusable looping player; the loop spans the whole decoded stream.
	var loopLen int64
	if l, ok := stream.(interface{ Length() int64 }); ok {
		loopLen = l.Length()
	}
	p, err := context.NewPlayer(audio.NewInfiniteLoop(stream, loopLen))
	if err != nil {
		return err
	}
	p.SetVolume(b.volume)
	bgmPlayers[b.track] = p
	return nil
}

// loadSE decodes a one-shot effect, keeping the PCM so each play gets its own
// player (allowing the same SE to overlap).
func loadSE(s soundSpec, resource []byte) error {
	stream, err := decode(resource, s.fileType)
	if err != nil {
		return err
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		return err
	}
	seBytes[s.sound] = pcm
	seVolume[s.sound] = s.volume
	return nil
}

// PlaySound plays a one-shot effect: a fresh, overlapping player. Unloaded
// sounds are a no-op.
func PlaySound(s Sound) {
	pcm := seBytes[s]
	if pcm == nil {
		return
	}
	p := context.NewPlayerFromBytes(pcm)
	p.SetVolume(seVolume[s])
	p.Play() // GC'd once playback finishes
}

// PlayBGM plays the given looping track. Re-requesting the track that is already
// playing is a no-op, so scenes that share a track (opening↔title,
// in-game↔result) keep the music seamless; switching tracks stops the previous
// one and starts the new from the top.
func PlayBGM(track BGMTrack) {
	if bgmPlaying && currentBGM == track {
		return
	}
	if cur := bgmPlayers[currentBGM]; cur != nil {
		cur.Pause()
	}
	currentBGM = track
	p := bgmPlayers[track]
	if p == nil {
		bgmPlaying = false
		return
	}
	if err := p.Rewind(); err != nil {
		log.Println(err)
	}
	p.Play()
	bgmPlaying = true
}

// StopBGM pauses the current background track.
func StopBGM() {
	if cur := bgmPlayers[currentBGM]; cur != nil {
		cur.Pause()
	}
	bgmPlaying = false
}
