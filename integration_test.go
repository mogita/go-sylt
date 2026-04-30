package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	id3v2 "github.com/bogem/id3v2/v2"
)

// minimalMP3Header is a valid empty ID3v2.4 tag header (10 bytes).
// id3v2.Open accepts this and Save() will rewrite the file with new frames.
var minimalMP3Header = []byte{'I', 'D', '3', 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

func writeMinimalMP3(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, minimalMP3Header, 0644); err != nil {
		t.Fatalf("failed to create test mp3: %v", err)
	}
}

func TestProcessFiles_OriginalUnmodified(t *testing.T) {
	tempDir := t.TempDir()
	mp3Path := filepath.Join(tempDir, "song.mp3")
	writeMinimalMP3(t, mp3Path)

	originalBytes, err := os.ReadFile(mp3Path)
	if err != nil {
		t.Fatalf("failed to read original: %v", err)
	}

	lyricsPath := filepath.Join(tempDir, "song.lrc")
	if err := os.WriteFile(lyricsPath, []byte("[00:01.00]Hello\n[00:02.00]World"), 0644); err != nil {
		t.Fatalf("failed to write lyrics: %v", err)
	}

	if err := processFiles(mp3Path, lyricsPath, "eng"); err != nil {
		t.Fatalf("processFiles failed: %v", err)
	}

	afterBytes, err := os.ReadFile(mp3Path)
	if err != nil {
		t.Fatalf("failed to read original after processing: %v", err)
	}
	if !bytes.Equal(originalBytes, afterBytes) {
		t.Errorf("original MP3 was modified by processFiles; want unchanged")
	}

	tag, err := id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	if err != nil {
		t.Fatalf("failed to reopen original: %v", err)
	}
	defer tag.Close()
	if frames := tag.GetFrames("SYLT"); len(frames) != 0 {
		t.Errorf("original has %d SYLT frames; want 0", len(frames))
	}
}

func TestProcessFiles_LargeFileStreaming(t *testing.T) {
	tempDir := t.TempDir()
	mp3Path := filepath.Join(tempDir, "big.mp3")

	// Build a 5 MB "MP3": valid ID3v2 header + 5 MB of zero-padding (audio frames).
	// We don't need real MP3 audio frames — id3v2 only reads the tag area.
	const padSize = 5 * 1024 * 1024
	f, err := os.Create(mp3Path)
	if err != nil {
		t.Fatalf("failed to create big mp3: %v", err)
	}
	if _, err := f.Write(minimalMP3Header); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := f.Write(make([]byte, padSize)); err != nil {
		t.Fatalf("write padding: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	lyricsPath := filepath.Join(tempDir, "big.lrc")
	if err := os.WriteFile(lyricsPath, []byte("[00:01.00]X"), 0644); err != nil {
		t.Fatalf("write lyrics: %v", err)
	}

	if err := processFiles(mp3Path, lyricsPath, "eng"); err != nil {
		t.Fatalf("processFiles failed: %v", err)
	}

	outPath := getOutputPath(mp3Path)
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	// Output should be roughly the same size as input plus the small SYLT frame.
	if info.Size() < padSize {
		t.Errorf("output size %d too small; want >= %d", info.Size(), padSize)
	}
}
