package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var lang string
	flag.StringVar(&lang, "lang", "und", "3-letter ISO 639-2 language code")
	flag.Parse()

	args := flag.Args()

	switch len(args) {
	case 0:
		fmt.Fprintf(os.Stderr, "Usage: %s [--lang <code>] <mp3_file> [lyrics_file]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  With lyrics_file: Add SYLT lyrics to MP3 file\n")
		fmt.Fprintf(os.Stderr, "  Without lyrics_file: Read and display existing SYLT lyrics from MP3 file\n")
		os.Exit(1)
	case 1:
		// Read and display existing SYLT from MP3 file
		mp3File := args[0]
		if err := readSYLT(mp3File); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case 2:
		// Add SYLT lyrics to MP3 file
		mp3File := args[0]
		lyricsFile := args[1]

		if err := validateLanguageCode(lang); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := processFiles(mp3File, lyricsFile, lang); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Successfully added SYLT lyrics to MP3 file")
	default:
		fmt.Fprintf(os.Stderr, "Error: too many arguments\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [--lang <code>] <mp3_file> [lyrics_file]\n", os.Args[0])
		os.Exit(1)
	}
}
