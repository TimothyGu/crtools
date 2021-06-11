// Command remove_disallow expands and inlines various DISALLOW_* macros.
// See https://crbug.com/1010217 for why this is useful.
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

type Change struct {
	Action   Action
	From, To int
	Data     string
}

type Action int

const (
	Add Action = iota + 1
	Rem
)

type scannerState struct {
	startOfIndent []int
	labels        map[int][]int
	changes       []Change
}

func scan(r io.Reader) (string, error) {
	s := bufio.NewScanner(r)
	st := scannerState{
		labels:        make(map[int][]int),
		startOfIndent: []int{0},
	}
	var lines []string
	var lineStuff []Line
	for lineNum := 0; s.Scan(); lineNum++ {
		line := s.Text()
		l := detect(line)
		ind := numIndent(line)
		lines = append(lines, line)
		lineStuff = append(lineStuff, l)

		if l.Type == Empty || l.Type == PreprocessorDirective {
			// log.Printf("line %d(%d): empty or preproc", lineNum+1, ind)
			continue
		}
		if ind >= len(st.startOfIndent)+2 {
			// log.Printf("line %d(%d): too much indentation", lineNum+1, ind)
			continue // too much indentation, must be line continuation
		}

		for ind >= len(st.startOfIndent) {
			st.startOfIndent = append(st.startOfIndent, lineNum)
		}
		if ind < len(st.startOfIndent) {
			st.startOfIndent = st.startOfIndent[:ind+1]
			st.startOfIndent[ind] = lineNum
		}

		if l.Type == Label {
			minusOne := st.startOfIndent[ind-1]
			minusOneData := lineStuff[minusOne]
			if minusOneData.Type == ClassDecl {
				st.labels[minusOne] = append(st.labels[minusOne], lineNum)
			}
		}
	}
	if s.Err() != nil {
		return "", s.Err()
	}

	st.startOfIndent = []int{0}
	for lineNum, line := range lines {
		l := lineStuff[lineNum]
		ind := numIndent(line)
		if l.Type == Empty || l.Type == PreprocessorDirective {
			continue // empty line
		}
		if ind >= len(st.startOfIndent)+2 {
			continue // too much indentation, must be line continuation
		}

		for ind >= len(st.startOfIndent) {
			st.startOfIndent = append(st.startOfIndent, lineNum)
		}
		if ind < len(st.startOfIndent) {
			st.startOfIndent = st.startOfIndent[:ind+1]
			st.startOfIndent[ind] = lineNum
		}

		if l.Type == DisallowCopy ||
			l.Type == DisallowAssign ||
			l.Type == DisallowCopyAndAssign ||
			l.Type == DisallowImplicitConstructors {
			log.Printf("found %s(%s) (line %d)", l.Type, l.Data, lineNum+1)
			if ind < 2 {
				log.Printf("apparently no home")
			} else {
				minusTwo := st.startOfIndent[ind-2]
				minusTwoData := lineStuff[minusTwo]
				log.Printf("ind-2 type: %s(%s) (line %d)", minusTwoData.Type, minusTwoData.Data, minusTwo+1)

				if minusTwoData.Type != ClassDecl {
					return "", fmt.Errorf("%s(%s) (line %d) mismatches %s(%s) (line %d)", l.Type, l.Data, lineNum+1, minusTwoData.Type, minusTwoData.Data, minusTwo+1)
				} else if l.Data != minusTwoData.Data {
					return "", fmt.Errorf("%s(%s) (line %d) != %s(%s) (line %d)", l.Type, l.Data, lineNum+1, minusTwoData.Type, minusTwoData.Data, minusTwo+1)
				}

				minusOne := st.startOfIndent[ind-1]
				minusOneData := lineStuff[minusOne]
				log.Printf("ind-1 type: %s(%s) (line %d)", minusOneData.Type, minusOneData.Data, minusOne+1)

				firstLabel := -1
				firstPublic := -1
				for _, labelLine := range st.labels[minusTwo] {
					data := lineStuff[labelLine]
					log.Printf("eligible: %s(%s) (line %d)", data.Type, data.Data, labelLine+1)
					if data.Type == Label {
						if firstLabel < 0 {
							firstLabel = labelLine
						}
						if data.Data == "public" && firstPublic < 0 {
							firstPublic = labelLine
						}
					}
				}

				st.changes = append(st.changes, Change{
					Action: Rem,
					From:   lineNum,
				})

				indentStr := strings.Repeat(" ", ind)
				if firstPublic >= 0 {
					st.changes = append(st.changes, Change{
						Action: Add,
						To:     firstPublic + 1,
						Data:   fmt.Sprintf(replacements[l.Type], l.Data, indentStr),
					})
				} else if firstLabel >= 0 {
					st.changes = append(st.changes, Change{
						Action: Add,
						To:     firstLabel,
						Data:   fmt.Sprintf("%[3]spublic:\n"+replacements[l.Type], l.Data, indentStr, indentStr[:ind-1]),
					})
				} else {
					st.changes = append(st.changes, Change{
						Action: Add,
						To:     minusTwo + 1,
						Data:   fmt.Sprintf("%[3]spublic:\n"+replacements[l.Type], l.Data, indentStr, indentStr[:ind-1]),
					})
				}
			}
		}
	}

	if len(st.changes) == 0 {
		return "", nil
	}

	const remSentinel = "!!!REM!!!"
	for _, chg := range st.changes {
		switch chg.Action {
		case Rem:
			lines[chg.From] = remSentinel
		case Add:
			if lines[chg.To] == remSentinel {
				lines[chg.To] = chg.Data
			} else {
				lines[chg.To] = chg.Data + "\n" + lines[chg.To]
			}
		}
	}
	var builder strings.Builder
	for _, line := range lines {
		if line == remSentinel {
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

type LineType int

//go:generate go run golang.org/x/tools/cmd/stringer -type LineType
const (
	Unrelated LineType = iota
	Empty
	PreprocessorDirective
	DisallowCopy
	DisallowAssign
	DisallowCopyAndAssign
	DisallowImplicitConstructors
	ClassDecl
	Label
)

type Line struct {
	Type LineType
	Data string
}

var replacements = [...]string{
	DisallowCopy:   `%[2]s%[1]s(const %[1]s&) = delete;`,
	DisallowAssign: `%[2]s%[1]s& operator=(const %[1]s&) = delete;`,
	DisallowCopyAndAssign: `%[2]s%[1]s(const %[1]s&) = delete;
%[2]s%[1]s& operator=(const %[1]s&) = delete;`,
	DisallowImplicitConstructors: `%[2]s%[1]s() = delete;
%[2]s%[1]s(const %[1]s&) = delete;
%[2]s%[1]s& operator=(const %[1]s&) = delete;`,
}

var (
	isDisallowCopy          = regexp.MustCompile(`^ *DISALLOW_COPY\(([^)]*)\);`)
	isDisallowAssign        = regexp.MustCompile(`^ *DISALLOW_ASSIGN\(([^)]*)\);`)
	isDisallowCopyAndAssign = regexp.MustCompile(`^ *DISALLOW_COPY_AND_ASSIGN\(([^)]*)\);`)
	isDisallowImplicitCtors = regexp.MustCompile(`^ *DISALLOW_IMPLICIT_CONSTRUCTORS\(([^)]*)\);`)
	isEnumClass             = regexp.MustCompile(`\benum class\b`)
	extractClassDeclFinal   = regexp.MustCompile(`\bclass\b(?: [^:]*)? (?:\w+::)*([^: ]*) final(?:$| (?::.*)?)`)
	extractClassDecl        = regexp.MustCompile(`\bclass\b(?: [^:]*)? (?:\w+::)*([^: {]*)(?:$| {)`)
	extractLabel            = regexp.MustCompile(`^ *(\w+):$`)
	isPreprocDirective      = regexp.MustCompile(`^ *#`)
)

func detect(line string) Line {
	if isPreprocDirective.MatchString(line) {
		return Line{Type: PreprocessorDirective}
	} else if m := isDisallowCopy.FindStringSubmatch(line); m != nil {
		return Line{Type: DisallowCopy, Data: m[1]}
	} else if m := isDisallowAssign.FindStringSubmatch(line); m != nil {
		return Line{Type: DisallowAssign, Data: m[1]}
	} else if m := isDisallowCopyAndAssign.FindStringSubmatch(line); m != nil {
		return Line{Type: DisallowCopyAndAssign, Data: m[1]}
	} else if m := isDisallowImplicitCtors.FindStringSubmatch(line); m != nil {
		return Line{Type: DisallowImplicitConstructors, Data: m[1]}
	} else if isEnumClass.MatchString(line) {
		return Line{Type: Unrelated}
	} else if m := extractClassDeclFinal.FindStringSubmatch(line); m != nil {
		if m[1] == "CORE_EXPORT" || m[1] == "final" || m[1] == "{" {
			log.Fatalf("bad class name (final): %s from %s", m[1], line)
		}
		return Line{Type: ClassDecl, Data: m[1]}
	} else if m := extractClassDecl.FindStringSubmatch(line); m != nil {
		if m[1] == "CORE_EXPORT" || m[1] == "final" || m[1] == "{" {
			log.Fatalf("bad class name: %s from %s", m[1], line)
		}
		return Line{Type: ClassDecl, Data: m[1]}
	} else if m := extractLabel.FindStringSubmatch(line); m != nil {
		return Line{Type: Label, Data: m[1]}
	} else if numIndent(line) == -1 {
		return Line{Type: Empty}
	} else {
		return Line{Type: Unrelated}
	}
}

func numIndent(line string) int {
	for i, r := range line {
		if r != ' ' {
			return i
		}
	}
	return -1
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
