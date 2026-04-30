<p align="center">
  <h3 align="center">GO SYLT</h3>
  <p align="center">A pretty convenient CLI tool that reads and writes synced lyrics to MP3 files in SYLT (SYnchronized Lyrics/Text) format.</p>
  <p align="center">
    <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
    <a href="https://goreportcard.com/report/github.com/mogita/go-sylt"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/mogita/go-sylt?style=flat-square"></a>
    <a href="https://coveralls.io/github/mogita/go-sylt?branch=main"><img alt="Coverage Status" src="https://img.shields.io/coveralls/mogita/go-sylt/main.svg?style=flat-square"></a>
  </p>
</p>

## Features

- Supported input formats: LRC, SRT, and VTT
- Configurable language codes (case-insensitive 3-letter ISO 639-2 codes, e.g. `eng`, `zho`, `und`)
- Read and display existing SYLT lyrics from MP3 files
- Creates output files with " - sylt" suffix to preserve originals

> Still, backing up originals is recommended.

## Download

Visit the [Releases](https://github.com/mogita/go-sylt/releases) page for pre-compiled binaries.

You can also build from source:

```bash
git clone https://github.com/mogita/go-sylt
cd go-sylt
go build
```

## Usage

```bash
go-sylt [--lang <code>] <mp3_file> [lyrics_file]
```

> Currently `go-sylt` supports ID3v2.3 and ID3v2.4 tags only. For MP3 files with older or unsupported ID3 versions, please convert them to a supported version first. You can use tools like [Kid3](https://kid3.kde.org) for this purpose.

### Examples

```bash
# Write SYLT lyrics to MP3 file
./go-sylt song.mp3 lyrics.lrc

# Write lyrics and specify English language
./go-sylt --lang eng song.mp3 lyrics.srt

# Write lyrics using VTT format in Chinese
./go-sylt --lang zho song.mp3 subtitles.vtt

# Read existing SYLT lyrics from MP3 file and display
./go-sylt song.mp3
```

### Language codes

`--lang` accepts any 3-letter ISO 639-2 code and is case-insensitive (`ENG`, `Eng`, and `eng` are all accepted and stored as `eng`). Common values:

- `eng` — English
- `zho` — Chinese
- `jpn` — Japanese
- `kor` — Korean
- `spa` — Spanish
- `fra` — French
- `deu` — German
- `und` — undetermined (default)

For the full list, see the [ISO 639-2 language code list](https://www.loc.gov/standards/iso639-2/php/code_list.php).

## Output

The tool creates a new MP3 file with the same name as the input file but with " - sylt" appended before the extension. For example:

- Input: `song.mp3` → Output: `song - sylt.mp3`

## Credits

- [github.com/bogem/id3v2/v2](https://github.com/bogem/id3v2) - ID3v2 tag manipulation


## Contributing

Contributions are welcome! Should you find any issues or have any suggestions, kindly submit an issue or PR with the provided templates. Thank you!

## Development

- Go 1.21 or above
- [golangci-lint](https://golangci-lint.run/) for linting

```bash
# Run tests
go test -v

# Run linter
golangci-lint run
```

## Releasing

Visit the [Releases](https://github.com/mogita/go-sylt/releases) page, create a release on the `main` branch with a tag in the format of `X.Y.Z` or `X.Y.Z-rc1`. The workflow will automatically build and upload the binaries.

## License

MIT © [mogita](https://github.com/mogita)