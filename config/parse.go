package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Parsing errors
var (
	ErrInvalidKey = errors.New("config: invalid key")
	ErrUnknownKey = errors.New("config: unknown key")
	ErrParse      = errors.New("config: parse error")
	ErrArray      = errors.New("config: array unclosed")
)

func parseArray(s *bufio.Scanner, l *int, trail string) ([]string, error) {
	var ret []string
	var err error

	switch trail {
	case "[":
		ret, err = parseArrayBody(s, l)
	case "]":
		err = errors.New("invalid array syntax")
	case "[]":
		ret = nil

	default:
		ret = make([]string, 1)
		ret[0] = trail
	}

	return ret, err
}

func parseArrayBody(s *bufio.Scanner, l *int) ([]string, error) {
	ret := make([]string, 0, 2)

scanloop:
	for s.Scan() {
		*l++

		str := s.Text()
		switch str {
		case "[":
			continue scanloop
		case "]":
			break scanloop
		case "":
			return ret, errors.New("unclosed array braces")
		}

		ret = append(ret, strings.TrimSpace(str))
	}
	if s.Err() != nil {
		return nil, s.Err()
	}

	if len(ret) == 0 {
		ret = nil
	}
	return ret, nil
}

// parse advances through the config file, extracting one config key per line.
// Keys are separated from content by a single ':' (colon). Any UTF-8 text can
// be placed around the colon and will be handled.
func parse(c *Config, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config failed: %w", err)
	}

	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for num := 1; s.Scan(); num++ {
		l := s.Text()
		str := strings.SplitN(l, ":", 2)

		// Probably blank line
		if len(str) != 2 {
			continue
		}
		// Comment; ignore
		if strings.HasPrefix(l, "//") {
			continue
		}

		var err error
		key, trail := strings.ToLower(str[0]), strings.TrimSpace(str[1])
		switch key {
		case "addr":
			c.ListenAddr = trail
		case "port":
			c.ListenPort, err = strconv.ParseUint(trail, 10, 64)
		case "proxies":
			c.TrustedProxies, err = parseArray(s, &num, trail)
		case "game_timeout":
			i, e := strconv.ParseInt(trail, 10, 32)
			c.GameTimeout, err = time.Second*time.Duration(i), e
		default:
			err = fmt.Errorf("unknown key: %q", key)
		}

		// Parsing error from last pass
		if err != nil {
			return fmt.Errorf("config: parse error: %w: line %d", err, num)
		}
	}

	return nil
}
