// Package config represents an abstraction over the Gahoot config file and
// implements a parser, loader and validator for said configuration.
//
// Configuration consists of a simple text file, which contains a set of keys
// defined via "key: value", where value may be blank. Trailing whitespace
// before and after the "value" will be stripped, but any other valid UTF-8
// characters may be otherwise
package config

import (
	"fmt"

	"github.com/go-playground/validator"
)

type Config struct {
	validator *validator.Validate

	ListenAddr string
	ListenPort uint64
}

// FullAddr returns the full address for use in serving based on both
// ListenAddr and ListenPort in the format expected by net.Dial
func (c Config) FullAddr() string {
	return fmt.Sprintf("%s:%d", c.ListenAddr, c.ListenPort)
}

func New(path string) (Config, error) {
	c := Config{
		validator: validator.New(),
	}
	err := parse(&c, path)

	return c, err
}
