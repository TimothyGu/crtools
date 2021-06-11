// Command remove_base_macros removes unneeded "base/macros.h" includes
// from input files.
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

const includeBaseMacros = `#include "base/macros.h"`

var baseMacroUsage = regexp.MustCompile(`\b(?:ignore_result|DISALLOW_(?:COPY|ASSIGN|COPY_AND_ASSIGN|IMPLICIT_CONSTRUCTORS))\b`)

func scan(r io.Reader) (string, error) {
	s := bufio.NewScanner(r)
	var lines []string
	seenInclude, seenUsage := false, false
	for s.Scan() {
		line := s.Text()
		lines = append(lines, line)
		seenInclude = seenInclude || line == includeBaseMacros
		seenUsage = seenUsage || baseMacroUsage.MatchString(line)
	}
	if !seenInclude || seenUsage || s.Err() != nil {
		return "", s.Err()
	}

	var builder strings.Builder
	for _, line := range lines {
		if line == includeBaseMacros {
			continue
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}

func readAndScan(name string) error {
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	replacement, err := scan(f)
	if err != nil {
		return err
	}
	if len(replacement) > 0 {
		n, err := f.WriteAt([]byte(replacement), 0)
		if err != nil {
			return err
		}
		err = f.Truncate(int64(n))
	}
	return err
}

func main() {
	if len(os.Args) <= 1 {
		replacement, err := scan(os.Stdin)
		if err != nil {
			panic(err)
		}
		if len(replacement) > 0 {
			_, err := io.WriteString(os.Stdout, replacement)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Fprintln(os.Stderr, "no changes")
		}
		return
	}

	for _, name := range os.Args[1:] {
		log.Println("scanning", name)
		if err := readAndScan(name); err != nil {
			panic(fmt.Errorf("%s: %w", name, err))
		}
	}
}
