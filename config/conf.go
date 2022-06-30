// Package config represents an abstraction over the Gahoot config file and
// implements a parser, loader and validator for said configuration.
//
// Configuration consists of a simple text file, which contains a set of keys
// defined via "key: value", where value may be blank. Trailing whitespace
// before and after the "value" will be stripped, but any other valid UTF-8
// characters may be otherwise.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator"
)

type Config struct {
	ListenAddr     string   `validate:"ip_addr|hostname"`
	ListenPort     uint64   `validate:"gte=1,lte=65535"`
	TrustedProxies []string `validate:"dive,ip_addr"`

	GameTimeout time.Duration
}

// FullAddr returns the full address for use in serving based on both
// ListenAddr and ListenPort in the format expected by net.Dial.
func (c Config) FullAddr() string {
	return fmt.Sprintf("%s:%d", c.ListenAddr, c.ListenPort)
}

func New(path string, validator *validator.Validate) (Config, error) {
	c := Config{}
	err := parse(&c, path)
	if err != nil {
		return c, err
	}

	err = validator.Struct(c)
	if err != nil {
		return c, err
	}

	return c, nil
}

// FormatErrors returns configuration errors formatted well for the user. Each
// error returned is prefaced with a tab ("\t") character, such that the error
// string can be shown in a block with a header, as is expected.
//
// If err is of type validator.ValidationErrors, the errors are formatted to
// show invalid keys and reasons for validation failure. If not, err.Error() is
// returned prefaced with a tab.
func FormatErrors(err error) string {
	e, ok := err.(validator.ValidationErrors)
	if !ok {
		return "\t" + err.Error()
	}

	str := &strings.Builder{}
	for _, elem := range e {
		fmt.Fprintf(str, "\tkey %q: %v: must be %s %s", elem.StructField(), elem.Value(), elem.Tag(), elem.Param())
	}

	return str.String()
}
