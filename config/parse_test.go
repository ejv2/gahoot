package config

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseArray(t *testing.T) {
	tests := []struct {
		src     string
		expects []string
		err     string
	}{
		{"[]", nil, ""},
		{"[\n]", nil, ""},
		{"", []string{""}, ""},
		{"[\n1.1.1.1\n]", []string{"1.1.1.1"}, ""},
		{"[\n1.1.1.1\n8.8.8.8\n]", []string{"1.1.1.1", "8.8.8.8"}, ""},

		{"1.1.1.1", []string{"1.1.1.1"}, ""},
		{"[1.1.1.1]", []string{"[1.1.1.1]"}, ""}, // NOTE: Notice lack of newlines!

		{"]", []string{}, "invalid array syntax"},
		{"[\n\n]", []string{}, "unclosed array"},
		{"[\n\n", []string{}, "unclosed array"},
		{"[\n\n1.1.1.1]", []string{}, "unclosed array"},
		{"[\n1.1.1.1\n\n8.8.8.8]", []string{"1.1.1.1"}, "unclosed array"},
		{"[\n1.1.1.1\n\n", []string{"1.1.1.1"}, "unclosed array"},
	}

	for _, elem := range tests {
		r := strings.NewReader(elem.src)
		br := bufio.NewScanner(r)

		tmp := 0

		br.Scan()
		trail := strings.SplitN("example:"+br.Text(), ":", 2)[1]
		arr, err := parseArray(br, &tmp, trail)

		if elem.err == "" {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		} else {
			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), elem.err) {
				t.Errorf("unexpected or wrong error: expected %v, got %v", elem.err, err)
			}
		}

		if len(elem.expects) != len(arr) {
			t.Errorf("bad array parse: expected %v, got %v", elem.expects, arr)
			continue
		}
		for i, comp := range elem.expects {
			if i >= len(arr) {
				break
			}

			if comp != arr[i] {
				t.Errorf("bad array parse: expected %v, got %v [differ at %d]", elem.expects, arr, i)
			}
		}
	}
}
