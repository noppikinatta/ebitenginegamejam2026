.PHONY: gen run test test-cov build sound-gen sound-pak junk-img

gen:
	go generate ./...

# Regenerate the placeholder sound sources into asset/sound/raw (gitignored),
# then bundle them. Drop real .wav/.mp3/.ogg into asset/sound/raw instead to use
# licensed assets, then run `make sound-pak`.
sound-gen:
	mkdir -p asset/sound/raw
	go run tools/gensound/main.go asset/sound/raw
	$(MAKE) sound-pak

# Bundle asset/sound/raw/* into the committed, obfuscated asset/sound/sounds.pak.
sound-pak:
	go run tools/sndpak/main.go asset/sound/raw asset/sound/sounds.pak

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