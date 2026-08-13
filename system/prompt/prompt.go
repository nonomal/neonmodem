package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var (
	in  io.Reader = os.Stdin
	out io.Writer = os.Stdout

	scanner *bufio.Scanner

	readSecret = func() ([]byte, error) {
		return term.ReadPassword(int(syscall.Stdin))
	}
)

func lineScanner() *bufio.Scanner {
	if scanner == nil {
		scanner = bufio.NewScanner(in)
	}
	return scanner
}

func Line(question string, field string) (string, error) {
	s := lineScanner()

	for {
		fmt.Fprintf(out, "%s: ", question)

		if !s.Scan() {
			if err := s.Err(); err != nil {
				return "", fmt.Errorf("could not read %s: %w", field, err)
			}
			return "", fmt.Errorf("could not read %s: no input", field)
		}

		if answer := strings.ReplaceAll(s.Text(), " ", ""); answer != "" {
			return answer, nil
		}

		fmt.Fprintln(out, "Invalid input")
	}
}

func Secret(question string, field string) (string, error) {
	for {
		fmt.Fprintf(out, "%s (will not echo): ", question)

		answer, err := readSecret()
		fmt.Fprintln(out, "")
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", fmt.Errorf("could not read %s: no input", field)
			}
			return "", fmt.Errorf("could not read %s: %w", field, err)
		}

		if len(answer) > 0 {
			return string(answer), nil
		}

		fmt.Fprintln(out, "Invalid input")
	}
}
