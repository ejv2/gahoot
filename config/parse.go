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
)

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
		s := strings.SplitN(l, ":", 2)

		// Probably blank line
		if len(s) != 2 {
			continue
		}
		// Comment; ignore
		if strings.HasPrefix(l, "//") {
			continue
		}

		var err error
		key, trail := strings.ToLower(s[0]), strings.Trim(s[1], " \t")
		switch key {
		case "addr":
			c.ListenAddr = trail
		case "port":
			c.ListenPort, err = strconv.ParseUint(trail, 10, 64)
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
