.PHONY: gen run test test-cov build sound-gen sound-pak bgm-ogg junk-img proj-img bg-img

gen:
	go generate ./...

# Regenerate placeholder audio: per-weapon SE into asset/sound/raw (gitignored),
# the two BGM tracks into asset/sound/bgm_title.wav + bgm_game.wav (committed),
# then rebuild the SE pak from scratch (-rebuild, so removed effects don't linger).
# Drop real SE into asset/sound/raw and run `make sound-pak` (merge) to swap one;
# replace the bgm_*.wav files for the real (self-authored) BGM.
sound-gen:
	mkdir -p asset/sound/raw
	go run tools/gensound/main.go asset/sound/raw asset/sound
	go run tools/sndpak/main.go -rebuild asset/sound/raw asset/sound/se.pak

# Bundle the sound EFFECTS in asset/sound/raw/* into the committed, obfuscated
# asset/sound/se.pak. MERGE by default: files in raw/ override same-named entries
# in the existing pak, so you can partially replace SE (drop one file into raw/
# and repack) without needing every source. Use `-rebuild` to build only from
# raw/ (e.g. to drop an effect). BGM is committed directly and is not packed.
sound-pak:
	go run tools/sndpak/main.go asset/sound/raw asset/sound/se.pak

# Convert the committed BGM WAVs to Ogg Vorbis (smaller WASM payload) while
# preserving seamless looping: records each track's exact loop length into a
# <base>.ogg.loop sidecar and pads ~1s of silence before encoding so the lossy
# encoder's padding never bleeds into the loop region. Requires ffmpeg+libvorbis.
# After running, wire asset/sound.go to fileTypeOgg + the recorded loop lengths
# (see the printed snippet / the .ogg.loop files).
bgm-ogg:
	go run tools/wav2ogg/main.go asset/sound asset/sound/bgm_title.wav asset/sound/bgm_game.wav

# Regenerate the per-type junk placeholder images into asset/img.
junk-img:
	go run tools/genjunkimg/main.go asset/img

# Regenerate the cosmetic junk-projectile placeholder sprites into asset/img.
proj-img:
	go run tools/genprojimg/main.go asset/img

# Regenerate the placeholder scrolling background (seamless 1280x720) into asset/img.
bg-img:
	go run tools/genbgimg/main.go asset/img/background.png

run:
	go run app/main.go

test:
	go test -v ./...

test-cov:
	go test -cover -coverprofile=cover.out -v ./... && go tool cover -html=cover.out

build:
	GOOS=js GOARCH=wasm go build -o=release/game.wasm app/main.go