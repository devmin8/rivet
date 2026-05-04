package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func String(in io.Reader, out io.Writer, label string) (string, error) {
	fmt.Fprint(out, label)

	value, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	return strings.TrimSpace(value), nil
}

func Password(in *os.File, out io.Writer, label string) (string, error) {
	fmt.Fprint(out, label)

	password, err := term.ReadPassword(int(in.Fd()))
	fmt.Fprintln(out)
	if err != nil {
		return "", err
	}

	return string(password), nil
}
