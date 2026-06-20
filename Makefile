.PHONY: gen run test test-cov build sound-gen sound-pak junk-img

gen:
	go generate ./...

# Regenerate placeholder audio: SE into asset/sound/raw (gitignored), BGM into
# asset/sound/bgm.wav (committed), then bundle the SE. Drop real SE .wav/.mp3/.ogg
# into asset/sound/raw and run `make sound-pak`; replace asset/sound/bgm.wav for
# the real (self-authored) BGM.
sound-gen:
	mkdir -p asset/sound/raw
	go run tools/gensound/main.go asset/sound/raw asset/sound
	$(MAKE) sound-pak

# Bundle the sound EFFECTS in asset/sound/raw/* into the committed, obfuscated
# asset/sound/se.pak. BGM is committed directly and is not packed.
sound-pak:
	go run tools/sndpak/main.go asset/sound/raw asset/sound/se.pak

# Regenerate the per-type junk placeholder images into asset/img.
junk-img:
	go run tools/genjunkimg/main.go asset/img

run:
	go run app/main.go

test:
	go test -v ./...

test-cov:
	go test -cover -coverprofile=cover.out -v ./... && go tool cover -html=cover.out

build:
	GOOS=js GOARCH=wasm go build -o=release/game.wasm app/main.go