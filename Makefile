.PHONY: gen run test test-cov build sound-gen bgm-ogg junk-img proj-img bg-img meta-img

gen:
	go generate ./...

# Regenerate placeholder audio: per-weapon SE as loose wavs in asset/sound/se
# (committed, embedded directly) and the two BGM tracks into asset/sound/bgm.
# Existing wavs are SKIPPED so real audio is never clobbered; pass FORCE=1 to
# overwrite the placeholders. Swap a single SE by dropping a real wav of the
# same name into asset/sound/se.
sound-gen:
	mkdir -p asset/sound/se asset/sound/bgm
	go run tools/gensound/main.go $(if $(FORCE),-force) asset/sound/se asset/sound/bgm

# Convert the committed BGM WAVs to Ogg Vorbis (smaller WASM payload) while
# preserving seamless looping: records each track's exact loop length into a
# <base>.ogg.loop sidecar and pads ~1s of silence before encoding so the lossy
# encoder's padding never bleeds into the loop region. Requires ffmpeg+libvorbis.
# After running, wire asset/sound.go to fileTypeOgg + the recorded loop lengths
# (see the printed snippet / the .ogg.loop files).
bgm-ogg:
	go run tools/wav2ogg/main.go asset/sound/bgm asset/etc/bgm_title.wav asset/etc/bgm_game.wav

# Regenerate the per-type junk placeholder images into asset/img. Existing files
# are SKIPPED so real art is never clobbered; pass FORCE=1 to overwrite.
junk-img:
	go run tools/genjunkimg/main.go $(if $(FORCE),-force) asset/img

# Regenerate the cosmetic junk-projectile placeholder sprites into asset/img.
# Existing files are SKIPPED; pass FORCE=1 to overwrite.
proj-img:
	go run tools/genprojimg/main.go $(if $(FORCE),-force) asset/img

# Regenerate the placeholder scrolling background (seamless 1280x720) into asset/img.
# An existing file is SKIPPED; pass FORCE=1 to overwrite.
bg-img:
	go run tools/genbgimg/main.go $(if $(FORCE),-force) asset/img/background.png

# Regenerate the persistent-upgrade (workshop) placeholder icons (meta_*.png 24x24)
# and the coin sprite (coin.png 16x16) into asset/img. Existing files are SKIPPED;
# pass FORCE=1 to overwrite.
meta-img:
	go run tools/genmetaimg/main.go $(if $(FORCE),-force) asset/img

run:
	go run app/main.go

test:
	go test -v ./...

test-cov:
	go test -cover -coverprofile=cover.out -v ./... && go tool cover -html=cover.out

build:
	GOOS=js GOARCH=wasm go build -o=release/game.wasm app/main.go