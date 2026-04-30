package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestValidateLanguageCode(t *testing.T) {
	tests := []struct {
		lang    string
		wantErr bool
	}{
		{"eng", false},
		{"und", false},
		{"zho", false},
		{"EN", true},   // uppercase
		{"en", true},   // too short
		{"engl", true}, // too long
		{"e1g", true},  // contains number
		{"e-g", true},  // contains special char
		{"", true},     // empty
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			err := validateLanguageCode(tt.lang)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLanguageCode(%q) error = %v, wantErr %v", tt.lang, err, tt.wantErr)
			}
		})
	}
}

func TestParseLRC(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []LyricEntry
		wantErr bool
	}{
		{
			name: "valid LRC with 2-digit fractions",
			content: `[00:12.34]Line one
[00:25.67]Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
				{Text: "Line two", Ms: 25670},
			},
			wantErr: false,
		},
		{
			name: "valid LRC with 3-digit fractions",
			content: `[00:12.345]Line one
[00:25.678]Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12345},
				{Text: "Line two", Ms: 25678},
			},
			wantErr: false,
		},
		{
			name: "mixed fraction formats",
			content: `[00:12.34]Line one
[00:25.678]Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
				{Text: "Line two", Ms: 25678},
			},
			wantErr: false,
		},
		{
			name: "empty lines ignored",
			content: `[00:12.34]Line one
[00:25.67]
[00:30.00]Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
				{Text: "Line two", Ms: 30000},
			},
			wantErr: false,
		},
		{
			name:    "no valid entries",
			content: `invalid content`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "multiple timestamps on one line",
			content: `[00:10.00][00:20.00]Chorus line`,
			want: []LyricEntry{
				{Text: "Chorus line", Ms: 10000},
				{Text: "Chorus line", Ms: 20000},
			},
			wantErr: false,
		},
		{
			name: "mixed single and multi-timestamp lines",
			content: `[00:05.00]Intro
[00:10.00][00:20.00][00:30.00]Repeated`,
			want: []LyricEntry{
				{Text: "Intro", Ms: 5000},
				{Text: "Repeated", Ms: 10000},
				{Text: "Repeated", Ms: 20000},
				{Text: "Repeated", Ms: 30000},
			},
			wantErr: false,
		},
		{
			name:    "multiple timestamps with 3-digit fractions",
			content: `[00:01.500][00:02.500]X`,
			want: []LyricEntry{
				{Text: "X", Ms: 1500},
				{Text: "X", Ms: 2500},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLRC(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLRC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLRC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSRT(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []LyricEntry
		wantErr bool
	}{
		{
			name: "valid SRT",
			content: `1
00:00:12,340 --> 00:00:15,000
Line one

2
00:00:25,670 --> 00:00:28,000
Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
				{Text: "Line two", Ms: 25670},
			},
			wantErr: false,
		},
		{
			name: "multiline text",
			content: `1
00:00:12,340 --> 00:00:15,000
Line one
continues here`,
			want: []LyricEntry{
				{Text: "Line one continues here", Ms: 12340},
			},
			wantErr: false,
		},
		{
			name:    "no valid entries",
			content: `invalid content`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSRT(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSRT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSRT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVTT(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []LyricEntry
		wantErr bool
	}{
		{
			name: "valid VTT",
			content: `WEBVTT

00:12.340 --> 00:15.000
Line one

00:25.670 --> 00:28.000
Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
				{Text: "Line two", Ms: 25670},
			},
			wantErr: false,
		},
		{
			name:    "no valid entries",
			content: `WEBVTT`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "VTT with hours",
			content: `WEBVTT

01:00:12.340 --> 01:00:15.000
Line one

01:00:25.670 --> 01:00:28.000
Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 3612340},
				{Text: "Line two", Ms: 3625670},
			},
			wantErr: false,
		},
		{
			name: "VTT mixed hours and no-hours",
			content: `WEBVTT

00:12.340 --> 00:15.000
Line one

01:00:25.670 --> 01:00:28.000
Line two`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
				{Text: "Line two", Ms: 3625670},
			},
			wantErr: false,
		},
		{
			name: "VTT with extra whitespace around arrow",
			content: `WEBVTT

00:12.340  -->  00:15.000
Line one`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
			},
			wantErr: false,
		},
		{
			name: "VTT without WEBVTT header",
			content: `00:12.340 --> 00:15.000
Line one`,
			want: []LyricEntry{
				{Text: "Line one", Ms: 12340},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVTT(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVTT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVTT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLyrics(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filename string
		wantErr  bool
	}{
		{"LRC file", "[00:12.34]Test", "test.lrc", false},
		{"SRT file", "1\n00:00:12,340 --> 00:00:15,000\nTest", "test.srt", false},
		{"VTT file", "WEBVTT\n\n00:12.340 --> 00:15.000\nTest", "test.vtt", false},
		{"unsupported format", "content", "test.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLyrics(tt.content, tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLyrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildSYLT(t *testing.T) {
	entries := []LyricEntry{
		{Text: "Line one", Ms: 0},
		{Text: "Line two", Ms: 19680},
	}

	payload := buildSYLT(entries, "eng")

	// Check header
	if payload[0] != 0x03 {
		t.Errorf("Expected encoding 0x03, got 0x%02x", payload[0])
	}
	if string(payload[1:4]) != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", string(payload[1:4]))
	}
	if payload[4] != 0x02 {
		t.Errorf("Expected format 0x02, got 0x%02x", payload[4])
	}
	if payload[5] != 0x01 {
		t.Errorf("Expected type 0x01, got 0x%02x", payload[5])
	}
	if payload[6] != 0x00 {
		t.Errorf("Expected descriptor terminator 0x00, got 0x%02x", payload[6])
	}
}

func TestGetOutputPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/path/to/song.mp3", "/path/to/song - sylt.mp3"},
		{"song.mp3", "song - sylt.mp3"},
		{"/path/to/my song.mp3", "/path/to/my song - sylt.mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getOutputPath(tt.input)
			if got != tt.want {
				t.Errorf("getOutputPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSYLTFrame(t *testing.T) {
	tests := []struct {
		name      string
		frameData []byte
		wantErr   bool
		wantLang  string
		wantCount int
	}{
		{
			name: "valid SYLT frame UTF-8",
			frameData: func() []byte {
				// Build a valid SYLT frame
				entries := []LyricEntry{
					{Text: "Line one", Ms: 0},
					{Text: "Line two", Ms: 19680},
				}
				return buildSYLT(entries, "eng")
			}(),
			wantErr:   false,
			wantLang:  "eng",
			wantCount: 2,
		},
		{
			name: "valid SYLT frame UTF-16",
			frameData: func() []byte {
				// Manually build a UTF-16 SYLT frame
				buf := []byte{0x01}                 // UTF-16 encoding
				buf = append(buf, []byte("eng")...) // language
				buf = append(buf, 0x02)             // timestamp format: milliseconds
				buf = append(buf, 0x01)             // content type: lyrics
				buf = append(buf, 0x00, 0x00)       // empty descriptor (UTF-16 terminator)

				// Add UTF-16LE text entries
				buf = append(buf, []byte("L\x00i\x00n\x00e\x00 \x00o\x00n\x00e\x00")...) // "Line one" in UTF-16LE
				buf = append(buf, 0x00, 0x00)                                            // UTF-16 terminator
				buf = append(buf, 0x00, 0x00, 0x00, 0x00)                                // timestamp 0

				buf = append(buf, []byte("L\x00i\x00n\x00e\x00 \x00t\x00w\x00o\x00")...) // "Line two" in UTF-16LE
				buf = append(buf, 0x00, 0x00)                                            // UTF-16 terminator
				buf = append(buf, 0x00, 0x00, 0x4c, 0xe0)                                // timestamp 19680

				return buf
			}(),
			wantErr:   false,
			wantLang:  "eng",
			wantCount: 2,
		},
		{
			name:      "frame too short",
			frameData: []byte{0x03, 0x65, 0x6e},
			wantErr:   true,
		},
		{
			name:      "unsupported timestamp format",
			frameData: []byte{0x03, 0x65, 0x6e, 0x67, 0x01, 0x01, 0x00},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, lang, err := parseSYLTFrame(tt.frameData)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSYLTFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if lang != tt.wantLang {
					t.Errorf("parseSYLTFrame() lang = %v, want %v", lang, tt.wantLang)
				}
				if len(entries) != tt.wantCount {
					t.Errorf("parseSYLTFrame() entries count = %v, want %v", len(entries), tt.wantCount)
				}
			}
		})
	}
}

func TestDecodeText(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		encoding byte
		want     string
		wantErr  bool
	}{
		{
			name:     "UTF-8",
			data:     []byte("Hello World"),
			encoding: 0x03,
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "Latin1",
			data:     []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, // "Hello"
			encoding: 0x00,
			want:     "Hello",
			wantErr:  false,
		},
		{
			name:     "UTF-16LE",
			data:     []byte{0x48, 0x00, 0x69, 0x00}, // "Hi" in UTF-16LE
			encoding: 0x01,
			want:     "Hi",
			wantErr:  false,
		},
		{
			name:     "UTF-16BE",
			data:     []byte{0x00, 0x48, 0x00, 0x69}, // "Hi" in UTF-16BE
			encoding: 0x02,
			want:     "Hi",
			wantErr:  false,
		},
		{
			name:     "UTF-16 with BOM LE",
			data:     []byte{0xFF, 0xFE, 0x48, 0x00, 0x69, 0x00}, // BOM + "Hi" in UTF-16LE
			encoding: 0x01,
			want:     "Hi",
			wantErr:  false,
		},
		{
			name:     "UTF-16 with BOM BE",
			data:     []byte{0xFE, 0xFF, 0x00, 0x48, 0x00, 0x69}, // BOM + "Hi" in UTF-16BE
			encoding: 0x01,
			want:     "Hi",
			wantErr:  false,
		},
		{
			name:     "UTF-16 too short",
			data:     []byte{0x48}, // Single byte
			encoding: 0x01,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "UTF-16BE odd length",
			data:     []byte{0x00, 0x48, 0x00}, // 3 bytes
			encoding: 0x02,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Unsupported encoding",
			data:     []byte("test"),
			encoding: 0xFF,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Empty data UTF-8",
			data:     []byte{},
			encoding: 0x03,
			want:     "",
			wantErr:  false,
		},
		{
			name:     "Empty data UTF-16",
			data:     []byte{},
			encoding: 0x01,
			want:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeText(tt.data, tt.encoding)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTextTerminator(t *testing.T) {
	tests := []struct {
		name     string
		encoding byte
		want     []byte
	}{
		{
			name:     "Latin1",
			encoding: 0x00,
			want:     []byte{0x00},
		},
		{
			name:     "UTF-16 with BOM",
			encoding: 0x01,
			want:     []byte{0x00, 0x00},
		},
		{
			name:     "UTF-16BE",
			encoding: 0x02,
			want:     []byte{0x00, 0x00},
		},
		{
			name:     "UTF-8",
			encoding: 0x03,
			want:     []byte{0x00},
		},
		{
			name:     "Unknown encoding",
			encoding: 0xFF,
			want:     []byte{0x00}, // fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTextTerminator(tt.encoding)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("getTextTerminator(%d) = %v, want %v", tt.encoding, got, tt.want)
			}
		})
	}
}

func TestSkipDescriptor(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		pos        int
		terminator []byte
		want       int
	}{
		{
			name:       "Single byte terminator found",
			data:       []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x57, 0x6f, 0x72, 0x6c, 0x64},
			pos:        0,
			terminator: []byte{0x00},
			want:       6,
		},
		{
			name:       "Double byte terminator found",
			data:       []byte{0x48, 0x00, 0x65, 0x00, 0x00, 0x00, 0x57, 0x00},
			pos:        0,
			terminator: []byte{0x00, 0x00},
			want:       5, // data is H\x00e\x00\x00\x00W\x00. Terminator \x00\x00 found at index 3. Returns 3+2=5
		},
		{
			name:       "Terminator not found",
			data:       []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f},
			pos:        0,
			terminator: []byte{0x00},
			want:       5,
		},
		{
			name:       "Start from middle",
			data:       []byte{0x48, 0x65, 0x6c, 0x00, 0x6f},
			pos:        2,
			terminator: []byte{0x00},
			want:       4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipDescriptor(tt.data, tt.pos, tt.terminator)
			if got != tt.want {
				t.Errorf("skipDescriptor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadEncodedText(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		pos      int
		encoding byte
		wantText string
		wantPos  int
		wantErr  bool
	}{
		{
			name:     "UTF-8 text with terminator",
			data:     []byte("Hello\x00World"),
			pos:      0,
			encoding: 0x03,
			wantText: "Hello",
			wantPos:  6,
			wantErr:  false,
		},
		{
			name:     "UTF-16 text with terminator",
			data:     []byte{'H', 0x00, 'i', 0x00, 0x00, 0x00, 'W', 'o', 'r', 'l', 'd'},
			pos:      0,
			encoding: 0x01,
			wantText: "Hi",
			wantPos:  6,
			wantErr:  false,
		},
		{
			name:     "Text without terminator",
			data:     []byte("Hello"),
			pos:      0,
			encoding: 0x03,
			wantText: "Hello",
			wantPos:  5,
			wantErr:  false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			pos:      0,
			encoding: 0x03,
			wantText: "",
			wantPos:  0,
			wantErr:  true,
		},
		{
			name:     "Position at end of data",
			data:     []byte("Hello"),
			pos:      5,
			encoding: 0x03,
			wantText: "",
			wantPos:  5,
			wantErr:  true,
		},
		{
			name:     "Position beyond data",
			data:     []byte("Hello"),
			pos:      10,
			encoding: 0x03,
			wantText: "",
			wantPos:  10,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotPos, err := readEncodedText(tt.data, tt.pos, tt.encoding)
			if (err != nil) != tt.wantErr {
				t.Errorf("readEncodedText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotText != tt.wantText {
				t.Errorf("readEncodedText() text = %v, want %v", gotText, tt.wantText)
			}
			if gotPos != tt.wantPos {
				t.Errorf("readEncodedText() pos = %v, want %v", gotPos, tt.wantPos)
			}
		})
	}
}
