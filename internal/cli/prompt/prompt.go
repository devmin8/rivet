package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
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

func RequiredString(in io.Reader, out io.Writer, label string) (string, error) {
	value, err := String(in, out, label)
	if err != nil {
		return "", err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", strings.TrimSuffix(label, ": "))
	}

	return value, nil
}

func Port(in io.Reader, out io.Writer, label string) (uint32, error) {
	value, err := RequiredString(in, out, label)
	if err != nil {
		return 0, err
	}

	port, err := strconv.ParseUint(value, 10, 32)
	if err != nil || port == 0 || port > 65535 {
		return 0, errors.New("port must be a number between 1 and 65535")
	}

	return uint32(port), nil
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
