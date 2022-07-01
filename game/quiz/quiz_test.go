package quiz_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethanv2/gahoot/game/quiz"
)

// HashTests are calculated based on the final representation of the marshalled
// struct, which includes every field, even if empty.
var HashTests = [...]struct {
	Source, Hash string
}{
	{
		`{"title":"Quiz 2","description":"The second quiz"}`,
		"4784B66B01CB131D7178E54586B3F9626EDBB0F9A72A649BEE72D9C1BB8E729B",
	},
	{
		`{}`,
		"ED6F2A1EDE24786BE75E6C74A76D7C598EDF857EFFC0C781A452A9E01BDA3262",
	},
}

func generateLong(howLong int) string {
	s := `{"name": "`
	trailer := `"}`
	s += strings.Repeat("A", howLong-len(s)-len(trailer))
	s += trailer

	return s
}

func TestLoadQuiz(t *testing.T) {
	tests := []struct {
		Source   string
		ShouldOk bool
		Expect   string
	}{
		// Empty testing
		{"", false, "empty"},
		// Bad JSON testing
		{"a", false, "invalid"},
		{`{"title": 1234}`, false, "type"},
		{"{}", true, ""},
		// Length testing
		{generateLong(quiz.MaxQuizSize), true, ""},
		{generateLong(quiz.MaxQuizSize + 1), false, "too large"},
		{generateLong(quiz.MaxQuizSize * 2), false, "too large"},
		{generateLong(quiz.MaxQuizSize - 1), true, ""},
		{generateLong(quiz.MaxQuizSize / 2), true, ""},
	}

	for _, elem := range tests {
		sr := strings.NewReader(elem.Source)
		_, err := quiz.LoadQuiz(sr, quiz.SourceUpload)

		if elem.ShouldOk {
			if err != nil {
				t.Errorf("loadquiz: unexpected error: %s", err.Error())
			}
		} else {
			if err == nil {
				t.Errorf("loadquiz: expected error")
			} else if !strings.Contains(err.Error(), elem.Expect) {
				t.Errorf("loadquiz: wrong error: %s", err.Error())
			}
		}
	}
}

func TestHash(t *testing.T) {
	for _, elem := range HashTests {
		sr := strings.NewReader(elem.Source)
		q, err := quiz.LoadQuiz(sr, quiz.SourceUpload)

		if err != nil {
			t.Error("unexpected error:", err.Error())
		}

		if q.String() != elem.Hash {
			t.Errorf("invalid hash: got %s, expected %s", q.String(), elem.Hash)
		}
	}
}
