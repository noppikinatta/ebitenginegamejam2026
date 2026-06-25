# BGM の OGG 変換とシームレスループ

WASM 配信を軽くするため、BGM を WAV から **Ogg Vorbis** に変換する仕組みの解説。
ツール本体は [`tools/wav2ogg/main.go`](../tools/wav2ogg/main.go)、ランタイム側は
[`asset/sound.go`](../asset/sound.go) の `decodeBGM`。

---

## 1. なぜ OGG にするのか

BGM は生 PCM の WAV で、サイズが大きい（例：`bgm_game.wav` は約 14MB）。
itch.io は WASM を gzip 転送するが、**PCM 音楽は gzip がほとんど効かない**（実測で
14% しか縮まない）。一方 OGG は約 1/10 になるので、ロード時間に直結する。

ただし OGG はロッシー圧縮なので、**ループ再生をそのままやると継ぎ目でノイズが出る**。
そこを解決するのがこのツールの主目的。

---

## 2. ループの何が問題か

ゲームは `audio.NewInfiniteLoop(stream, loopLen)` でループする。`loopLen` は
「ここまで再生したら先頭へ戻る」バイト位置。

- **WAV** の場合：デコード後ストリームの `Length()` が正確な全長を返すので、
  `loopLen = Length()` で**サンプル単位ぴったり**にループでき、継ぎ目も無音。
- **OGG** の場合：ロッシーで、デコード長に**エンコーダのパディング（前後の余り
  サンプル）が含まれる**。`Length()` をそのまま使うと余白まで再生してから戻るため
  **プツッと鳴る**。

→ 対策は2つ：

1. **WAV 由来の正確なループ長を別途記録**し、OGG の `Length()` ではなく
   その値を `NewInfiniteLoop` に渡す。
2. OGG の**末尾に約1秒の無音を足してからエンコード**し、エンコーダのパディングが
   無音領域に収まるようにする。ループ領域 `[0, loopLen)` は無傷のまま。
   （ループは `loopLen` で先頭に戻るので、足した無音は再生されない。安全マージン。）

---

## 3. ループ長をどう計算するか（ランタイムと完全一致）

ゲームの audio context は **44100Hz**。実 BGM は **44100Hz / ステレオ / 16bit**。
ランタイムは `wav.DecodeWithSampleRate(44100, …)` で **44100→44100 にリサンプル**
してから長さを測る。ツールはこれを**バイト単位で同じに**再現する必要がある。

Ebiten の該当ロジック（`audio/internal/convert` の `Resampling.Length()`）：

```
length = int64(float64(size) * float64(to) / float64(from))   // 切り捨て
length = length / bytesPerSample * bytesPerSample              // フレーム整列
```

- `size` … デコード後（**16bit ステレオに変換済み**＝モノは2ch化、8bitは16bit化）の
  バイト長 ＝ `フレーム数 × 4`
- `from` … ソースのサンプルレート（44100）
- `to` … context レート（44100）
- `bytesPerSample` … 16bit ステレオ＝ **4 バイト**

ツールの `loopLengthBytes()` はこの式を**同じ float64 演算順**で再現している。
レートが 44100 のソースならリサンプル無し（`loopLen = フレーム数 × 4`）。

### 計算例（実 BGM）

| トラック | ソース | フレーム数 | loopLen（バイト） | ＝秒 @48k |
|---|---|---|---|---|
| `bgm_game`  | 44100/stereo/16bit | 3,472,968 | **15,120,404** | 78.752s |
| `bgm_title` | 44100/stereo/16bit |   524,156 |  **2,282,036** | 11.886s |

`bgm_game` の検算：`3,472,968 × 4 = 13,891,872`（デコード後バイト）→
`int64(13,891,872 × 44100 / 44100) = 15,120,404` → 4整列済み。

---

## 4. ツールの処理フロー（`tools/wav2ogg`）

```
入力 WAV (44100/stereo/16bit)
   │
   ├─ readWAV       … RIFF をパース。JUNK 等の不要チャンクはスキップ。
   │                  PCM/8or16bit/1or2ch のみ対応（Ebiten のデコーダ準拠）
   │
   ├─ loopLengthBytes … 上記の式で「ランタイムが見るループ長」を算出
   │
   ├─ padWithSilence … data チャンクの末尾に「無音秒数 × srcRate」フレーム分の
   │                    ゼロを追加した正規 WAV を生成（-silence、既定1.0秒）
   │
   ├─ encodeOgg      … ffmpeg にパイプして libvorbis でエンコード
   │                    -ar srcRate -ac srcChannels で**ソースのレート/ch を維持**
   │                    （理由は §5）。品質は -q（既定5）
   │
   └─ 出力:
        <base>.ogg        … 無音パッド込みの OGG
        <base>.ogg.loop   … ループ長（バイト・10進テキスト）
      ＋ 標準出力に貼り付け用 Go スニペットも表示
```

`-encode=false` でエンコードを省き、ループ長計算とパディング WAV 書き出し
（`-keep-wav`）だけ行える（ffmpeg の無いマシン用）。

**要件**：`ffmpeg`（libvorbis 入り）が PATH にあること。エンコード工程のみで使用。

---

## 5. なぜ OGG のレート/ch をソースのまま維持するのか

記録した `loopLen` は「**44100→44100 のリサンプル**を通した値」。
だから OGG も **44100/ステレオ**のままにして、ランタイムの
`vorbis.DecodeWithSampleRate(44100, …)` が**同じ 44100→44100 リサンプル**を行う
ようにする。こうすると WAV パスと OGG パスのリサンプル経路が一致し、記録した
`loopLen` がそのまま正しいループ点になる。

もし OGG を 44100 で書き出すと、ランタイムはリサンプルせず、`loopLen` の前提
（44100 起点の換算）とズレてしまう。

---

## 6. ゲーム側の読み込み（`asset/sound.go`）

BGM は `asset/sound/bgm/` を**ディレクトリごと `//go:embed`**している。
`decodeBGM(name)` がトラックごとに次の優先順で解決する：

```
1. <name>.ogg と <name>.ogg.loop が両方あり、loop長が正で、OGGがデコードできる
      → OGG を使い、loopLen は .ogg.loop の値を NewInfiniteLoop に渡す
2. それ以外（OGG 無し / loop 不正 / デコード失敗）
      → <name>.wav にフォールバック（Length() で正確ループ。従来挙動）
```

ディレクトリ埋め込みなので、**OGG を確認後に `bgm/*.wav` を消すだけで**
ogg-only 配信に切り替わる（コード変更不要、`.ogg`/`.loop` が残るのでループも維持）。

---

## 7. 運用手順

```bash
# 1. 変換（要 ffmpeg）。asset/sound/bgm/ に .ogg と .ogg.loop が出力される
make bgm-ogg

# 2. ローカルで起動し、ループの継ぎ目を一度試聴して確認
make run

# 3. 問題なければ .ogg と .ogg.loop をコミット → ゲームは OGG を使う

# 4. WASM からWAVを外して軽量化したくなったら（任意・最終段階）
rm asset/sound/bgm/*.wav   # .ogg/.loop が残るのでビルドもループも維持される
```

継ぎ目にクリックが残る場合の調整ノブ：

- `-silence`（無音秒数を増やす）：`go run tools/wav2ogg/main.go -silence 1.5 …`
- `-q`（Vorbis 品質を上げる、0..10）：`go run tools/wav2ogg/main.go -q 7 …`

> 補足：継ぎ目の連続性は理論上保たれるが、Vorbis デコーダの先頭プライミング
> 処理（granulepos）次第で稀にズレ得るので、差し替え後の試聴は1回しておくと安心。

---

## 8. 関連ファイル早見

| ファイル | 役割 |
|---|---|
| `tools/wav2ogg/main.go` | 変換ツール（ループ長算出・無音パッド・ffmpeg 呼び出し） |
| `asset/sound.go` (`decodeBGM`) | OGG優先・WAVフォールバックのBGMロード |
| `asset/sound/bgm/` | 埋め込まれる BGM ディレクトリ（wav / ogg / ogg.loop） |
| `Makefile` (`bgm-ogg`) | 変換のショートカット |
