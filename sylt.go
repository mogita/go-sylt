package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	id3v2 "github.com/bogem/id3v2/v2"
)

type LyricEntry struct {
	Text string
	Ms   uint32
}

// validateLanguageCode checks if the language code is a valid 3-letter ISO 639-2 code
func validateLanguageCode(lang string) error {
	if len(lang) != 3 {
		return fmt.Errorf("language code must be exactly 3 characters, got %d", len(lang))
	}
	// Basic validation - only lowercase letters
	for _, r := range lang {
		if r < 'a' || r > 'z' {
			return fmt.Errorf("language code must contain only lowercase letters")
		}
	}
	return nil
}

// parseLyrics detects format and parses lyrics file
func parseLyrics(content string, filename string) ([]LyricEntry, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".lrc":
		return parseLRC(content)
	case ".srt":
		return parseSRT(content)
	case ".vtt":
		return parseVTT(content)
	default:
		return nil, fmt.Errorf("unsupported file format: %s (supported: .lrc, .srt, .vtt)", ext)
	}
}

// parseLRC parses LRC format. Lines may contain one or more leading
// [mm:ss.xx] or [mm:ss.xxx] timestamps; each timestamp produces an entry
// pointing at the same text.
func parseLRC(content string) ([]LyricEntry, error) {
	var entries []LyricEntry
	tsRegex := regexp.MustCompile(`\[(\d{2}):(\d{2})\.(\d{2,3})\]`)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := tsRegex.FindAllStringSubmatchIndex(line, -1)
		if len(matches) == 0 {
			continue
		}

		// Verify all matches form a contiguous prefix at the start of the line.
		prefixEnd := 0
		valid := true
		for _, m := range matches {
			if m[0] != prefixEnd {
				valid = false
				break
			}
			prefixEnd = m[1]
		}
		if !valid {
			continue
		}

		text := strings.TrimSpace(line[prefixEnd:])
		if text == "" {
			continue
		}

		for _, m := range matches {
			minutes, _ := strconv.Atoi(line[m[2]:m[3]])
			seconds, _ := strconv.Atoi(line[m[4]:m[5]])
			fracStr := line[m[6]:m[7]]

			var milliseconds int
			if len(fracStr) == 2 {
				milliseconds, _ = strconv.Atoi(fracStr)
				milliseconds *= 10
			} else {
				milliseconds, _ = strconv.Atoi(fracStr)
			}

			totalMs := uint32((minutes*60+seconds)*1000 + milliseconds)
			entries = append(entries, LyricEntry{Text: text, Ms: totalMs})
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid LRC entries found")
	}
	return entries, nil
}

// parseSRT parses SRT subtitle format
func parseSRT(content string) ([]LyricEntry, error) {
	var entries []LyricEntry

	// Split content into blocks separated by double newlines
	blocks := regexp.MustCompile(`\n\s*\n`).Split(content, -1)
	timeRegex := regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3})\s*-->\s*\d{2}:\d{2}:\d{2},\d{3}`)

	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			continue
		}

		// Find the timestamp line
		var timeMatch []string
		var textLines []string
		for i, line := range lines {
			if timeMatch = timeRegex.FindStringSubmatch(line); timeMatch != nil {
				// Collect text lines after timestamp
				textLines = lines[i+1:]
				break
			}
		}

		if timeMatch == nil || len(textLines) == 0 {
			continue
		}

		hours, _ := strconv.Atoi(timeMatch[1])
		minutes, _ := strconv.Atoi(timeMatch[2])
		seconds, _ := strconv.Atoi(timeMatch[3])
		milliseconds, _ := strconv.Atoi(timeMatch[4])

		text := strings.TrimSpace(strings.Join(textLines, " "))
		if text == "" {
			continue
		}

		totalMs := uint32((hours*3600+minutes*60+seconds)*1000 + milliseconds)
		entries = append(entries, LyricEntry{Text: text, Ms: totalMs})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid SRT entries found")
	}
	return entries, nil
}

// parseVTT parses WebVTT format. Timestamps may be either mm:ss.mmm or
// hh:mm:ss.mmm and any amount of whitespace is permitted around the
// '-->' separator. The optional WEBVTT header line is skipped if present.
func parseVTT(content string) ([]LyricEntry, error) {
	var entries []LyricEntry
	lines := strings.Split(content, "\n")

	start := 0
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "WEBVTT") {
			start = i + 1
			break
		}
	}

	// Capture groups: 1=h? (optional), 2=mm, 3=ss, 4=mmm. End side ignored.
	// Per the WebVTT spec, minutes and seconds are exactly 2 digits and
	// milliseconds are exactly 3 digits; hours (when present) may be any
	// length.
	vttRegex := regexp.MustCompile(`(?:(\d+):)?(\d{2}):(\d{2})\.(\d{3})\s*-->\s*(?:\d+:)?\d{2}:\d{2}\.\d{3}`)

	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		matches := vttRegex.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}

		hours := 0
		if matches[1] != "" {
			hours, _ = strconv.Atoi(matches[1])
		}
		minutes, _ := strconv.Atoi(matches[2])
		seconds, _ := strconv.Atoi(matches[3])
		milliseconds, _ := strconv.Atoi(matches[4])

		text := ""
		j := i + 1
		for ; j < len(lines); j++ {
			nextLine := strings.TrimSpace(lines[j])
			if nextLine == "" {
				break
			}
			if text != "" {
				text += " "
			}
			text += nextLine
		}
		i = j // skip past consumed text lines so they aren't re-scanned

		if text == "" {
			continue
		}

		totalMs := uint32((hours*3600+minutes*60+seconds)*1000 + milliseconds)
		entries = append(entries, LyricEntry{Text: text, Ms: totalMs})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid VTT entries found")
	}
	return entries, nil
}

// buildSYLT creates SYLT frame payload
func buildSYLT(entries []LyricEntry, lang string) []byte {
	buf := make([]byte, 0, 128)
	buf = append(buf, 0x03)            // text encoding UTF-8
	buf = append(buf, []byte(lang)...) // language (3 bytes)
	buf = append(buf, 0x02)            // timestamp format: milliseconds
	buf = append(buf, 0x01)            // content type: lyrics
	buf = append(buf, 0x00)            // empty content descriptor (UTF-8 NUL)

	for _, entry := range entries {
		buf = append(buf, []byte(entry.Text)...)
		buf = append(buf, 0x00) // UTF-8 terminator
		ts := make([]byte, 4)
		binary.BigEndian.PutUint32(ts, entry.Ms)
		buf = append(buf, ts...)
	}
	return buf
}

// getOutputPath creates output filename with " - sylt" suffix
func getOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+" - sylt"+ext)
}

// processFiles handles the main logic
func processFiles(mp3File, lyricsFile, lang string) error {
	// Check if files exist
	if _, err := os.Stat(mp3File); os.IsNotExist(err) {
		return fmt.Errorf("MP3 file not found: %s", mp3File)
	}
	if _, err := os.Stat(lyricsFile); os.IsNotExist(err) {
		return fmt.Errorf("lyrics file not found: %s", lyricsFile)
	}

	// Read lyrics file
	content, err := os.ReadFile(lyricsFile)
	if err != nil {
		return fmt.Errorf("failed to read lyrics file: %v", err)
	}

	// Parse lyrics
	entries, err := parseLyrics(string(content), lyricsFile)
	if err != nil {
		return fmt.Errorf("failed to parse lyrics: %v", err)
	}

	// Open MP3 file to read tags
	tag, err := id3v2.Open(mp3File, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %v", err)
	}
	defer func() {
		if closeErr := tag.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close MP3 file: %v\n", closeErr)
		}
	}()

	// Build and add SYLT frame
	payload := buildSYLT(entries, lang)
	tag.AddFrame("SYLT", id3v2.UnknownFrame{Body: payload})

	// Generate output path and save
	outputPath := getOutputPath(mp3File)

	// Copy original file to new location first
	input, err := os.ReadFile(mp3File)
	if err != nil {
		return fmt.Errorf("failed to read original MP3 file: %v", err)
	}
	if err := os.WriteFile(outputPath, input, 0644); err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}

	// The tag will be closed by the defer function

	// Open the new file and add SYLT frame
	newTag, err := id3v2.Open(outputPath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open output MP3 file: %v", err)
	}
	defer func() {
		if closeErr := newTag.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close output MP3 file: %v\n", closeErr)
		}
	}()

	// Add SYLT frame to new file
	newTag.AddFrame("SYLT", id3v2.UnknownFrame{Body: payload})

	if err := newTag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 file: %v", err)
	}

	return nil
}

// parseSYLTFrame parses SYLT frame data and returns entries and language code
func parseSYLTFrame(frameData []byte) ([]LyricEntry, string, error) {
	if len(frameData) < 7 {
		return nil, "", fmt.Errorf("SYLT frame too short")
	}

	// Parse header
	encoding := frameData[0]
	lang := string(frameData[1:4])
	timestampFormat := frameData[4]
	if timestampFormat != 0x02 {
		return nil, "", fmt.Errorf("unsupported timestamp format: 0x%02x", timestampFormat)
	}

	contentType := frameData[5]
	if contentType != 0x01 {
		return nil, "", fmt.Errorf("unsupported content type: 0x%02x", contentType)
	}

	// Skip content descriptor based on encoding
	pos := 6
	terminator := getTextTerminator(encoding)
	pos = skipDescriptor(frameData, pos, terminator)
	if pos >= len(frameData) {
		return nil, "", fmt.Errorf("invalid SYLT frame: no descriptor terminator")
	}

	// Parse synchronized text entries
	var entries []LyricEntry
	for pos < len(frameData) {
		// Read text based on encoding
		text, newPos, err := readEncodedText(frameData, pos, encoding)
		if err != nil {
			break
		}
		pos = newPos

		// Read 4-byte timestamp
		if pos+4 > len(frameData) {
			break
		}
		timestamp := binary.BigEndian.Uint32(frameData[pos : pos+4])
		pos += 4

		if text != "" {
			entries = append(entries, LyricEntry{Text: text, Ms: timestamp})
		}
	}

	return entries, lang, nil
}

// getTextTerminator returns the terminator bytes for the given encoding
func getTextTerminator(encoding byte) []byte {
	switch encoding {
	case 0x00: // ISO-8859-1 (Latin1)
		return []byte{0x00}
	case 0x01: // UTF-16 with BOM
		return []byte{0x00, 0x00}
	case 0x02: // UTF-16BE without BOM
		return []byte{0x00, 0x00}
	case 0x03: // UTF-8
		return []byte{0x00}
	default:
		return []byte{0x00} // fallback to single byte
	}
}

// skipDescriptor skips the content descriptor field
func skipDescriptor(data []byte, pos int, terminator []byte) int {
	for pos < len(data) {
		if pos+len(terminator) <= len(data) {
			match := true
			for i, b := range terminator {
				if data[pos+i] != b {
					match = false
					break
				}
			}
			if match {
				return pos + len(terminator)
			}
		}
		pos++
	}
	return pos
}

// readEncodedText reads text based on the specified encoding
func readEncodedText(data []byte, pos int, encoding byte) (string, int, error) {
	terminator := getTextTerminator(encoding)

	// Find the end of the text
	textStart := pos

	// For UTF-16, we need to advance by 2 bytes at a time to maintain alignment
	step := 1
	if encoding == 0x01 || encoding == 0x02 {
		step = 2
	}

	for pos < len(data) {
		if pos+len(terminator) <= len(data) {
			match := true
			for i, b := range terminator {
				if data[pos+i] != b {
					match = false
					break
				}
			}
			if match {
				// Found terminator
				textBytes := data[textStart:pos]
				text, err := decodeText(textBytes, encoding)
				if err != nil {
					return "", pos, err
				}
				return text, pos + len(terminator), nil
			}
		}
		pos += step
	}

	// No terminator found, read to end
	if pos > textStart {
		textBytes := data[textStart:pos]
		text, err := decodeText(textBytes, encoding)
		if err != nil {
			return "", pos, err
		}
		return text, pos, nil
	}

	return "", pos, fmt.Errorf("no text found")
}

// decodeText decodes text bytes according to the specified encoding
func decodeText(data []byte, encoding byte) (string, error) {
	switch encoding {
	case 0x00: // ISO-8859-1 (Latin1)
		// Convert Latin1 to UTF-8
		runes := make([]rune, len(data))
		for i, b := range data {
			runes[i] = rune(b)
		}
		return string(runes), nil
	case 0x01: // UTF-16 with BOM
		if len(data) == 0 {
			return "", nil
		}
		if len(data) < 2 {
			return "", fmt.Errorf("UTF-16 data too short")
		}
		// Check for BOM and decode accordingly
		if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
			// Little endian BOM
			return decodeUTF16LE(data[2:])
		} else if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
			// Big endian BOM
			return decodeUTF16BE(data[2:])
		} else {
			// No BOM, assume little endian (common default)
			return decodeUTF16LE(data)
		}
	case 0x02: // UTF-16BE without BOM
		return decodeUTF16BE(data)
	case 0x03: // UTF-8
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported encoding: 0x%02x", encoding)
	}
}

// decodeUTF16LE decodes UTF-16 little endian bytes
func decodeUTF16LE(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	if len(data)%2 != 0 {
		return "", fmt.Errorf("invalid UTF-16LE data length")
	}

	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		if r != 0 { // Skip null characters
			runes = append(runes, r)
		}
	}
	return string(runes), nil
}

// decodeUTF16BE decodes UTF-16 big endian bytes
func decodeUTF16BE(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	if len(data)%2 != 0 {
		return "", fmt.Errorf("invalid UTF-16BE data length")
	}

	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i])<<8 | rune(data[i+1])
		if r != 0 { // Skip null characters
			runes = append(runes, r)
		}
	}
	return string(runes), nil
}

// readSYLT reads SYLT frames from an MP3 file
func readSYLT(mp3File string) error {
	// Check if file exists
	if _, err := os.Stat(mp3File); os.IsNotExist(err) {
		return fmt.Errorf("MP3 file not found: %s", mp3File)
	}

	// Open MP3 file
	tag, err := id3v2.Open(mp3File, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %v", err)
	}
	defer func() {
		if closeErr := tag.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close MP3 file: %v\n", closeErr)
		}
	}()

	// Get all SYLT frames
	syltFrames := tag.GetFrames("SYLT")
	if len(syltFrames) == 0 {
		return fmt.Errorf("no SYLT frames found in MP3 file")
	}

	// Process each SYLT frame
	for i, frame := range syltFrames {
		if unknownFrame, ok := frame.(id3v2.UnknownFrame); ok {
			entries, lang, err := parseSYLTFrame(unknownFrame.Body)
			if err != nil {
				fmt.Printf("Error parsing SYLT frame %d: %v\n", i+1, err)
				continue
			}

			if i > 0 {
				fmt.Println() // Add blank line between multiple frames
			}
			displaySYLT(entries, lang)
		}
	}

	return nil
}

// displaySYLT displays SYLT content in a readable format
func displaySYLT(entries []LyricEntry, lang string) {
	fmt.Printf("Language: %s\n", lang)
	for _, entry := range entries {
		minutes := entry.Ms / 60000
		seconds := (entry.Ms % 60000) / 1000
		milliseconds := entry.Ms % 1000
		fmt.Printf("[%02d:%02d.%03d] %s\n", minutes, seconds, milliseconds, entry.Text)
	}
}
