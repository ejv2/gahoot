package quiz_test

import (
	"strings"
	"testing"

	"github.com/ethanv2/gahoot/game/quiz"
)

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
