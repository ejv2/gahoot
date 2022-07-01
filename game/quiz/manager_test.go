package quiz

import (
	"os"
	"strings"
	"testing"
	"time"
)

var QuizFiles = [...]string{"loadfrom.gahoot", "duplicate.gahoot"}
var BadQuizFiles = [...]string{"malformed.gahoot", "notexist.gahoot"}

const (
	// Location to call LoadDir.
	QuizDir = "testdata" + string(os.PathSeparator) + "loaddir"
	// Expected number of loadable files.
	Contained  = 3
	ErrQuizDir = "testdata" + string(os.PathSeparator) + "errdir"
)

var QuizSources = [...]string{`{"title": "Quiz 2", "description": "The second quiz"}`}
var QuizTests = []Quiz{
	{
		Title:       "Quiz 1",
		Description: "The first quiz",
		Author:      "ethan_v2",
		Category:    "technology",
		Created:     time.Now(),
		Questions: []Question{
			{"First question", nil,
				[]Answer{
					{"1", false},
				}},
		},
	},
}

func init() {
	for _, elem := range QuizSources {
		q, err := LoadQuiz(strings.NewReader(elem), SourceFilesystem)
		if err != nil {
			panic(err)
		}
		QuizTests = append(QuizTests, q)
	}
}

func TestLoad(t *testing.T) {
	mgr := NewManager()
	for _, elem := range QuizTests {
		err := mgr.Load(elem)
		if err != nil {
			t.Errorf("unexpected load error: %s", err.Error())
		}
	}

	if len(mgr.qs) != len(QuizTests) {
		t.Errorf("expected %d items to be stored, got %d", len(QuizTests), len(mgr.qs))
	}
}

func TestLoadFrom(t *testing.T) {
	mgr := NewManager()

	for i, elem := range QuizFiles {
		q, err := mgr.LoadFrom("testdata" + string(os.PathSeparator) + elem)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
		}

		if q.source != SourceFilesystem {
			t.Errorf("wrong quiz source: should be SourceFilesystem (0), got %d", q.source)
		}
		if len(mgr.qs) != i+1 {
			t.Errorf("expected %d insertions, got %d", i+1, len(mgr.qs))
		}
	}
	if len(mgr.qs) != len(QuizFiles) {
		t.Errorf("wrong number of inserted quizzes: expected %d, got %d", len(QuizFiles), len(mgr.qs))
	}

	start := len(mgr.qs)
	for _, elem := range BadQuizFiles {
		_, err := mgr.LoadFrom("testdata" + string(os.PathSeparator) + elem)
		if err == nil {
			t.Error("bad quiz file: expected error, got nil")
		}
		if len(mgr.qs) > start {
			t.Errorf("inserted bad quiz: expected %d insertions, got %d", start, len(mgr.qs))
		}
	}
}

func TestLoadDir(t *testing.T) {
	mgr := NewManager()

	qs, err := mgr.LoadDir(QuizDir)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if len(qs) != Contained || len(mgr.qs) != Contained {
		t.Errorf("expected %d files loaded, got %d", Contained, len(qs))
	}

	before := len(mgr.qs)
	eqs, err := mgr.LoadDir(ErrQuizDir)
	if err == nil {
		t.Error("bad quiz dir: expected error, got nil")
	}
	if len(eqs) != 0 {
		t.Errorf("bad quiz dir: expected zero loads, got %d", len(eqs))
	}
	if before != len(mgr.qs) {
		t.Errorf("expected no change to quiz map, got %d change in length", before-len(mgr.qs))
	}
}

func TestGetAll(t *testing.T) {
	mgr := NewManager()
	for _, elem := range QuizTests {
		mgr.Load(elem)
	}

	if len(mgr.GetAll()) != len(QuizTests) {
		t.Errorf("getall: expected %d entries, returned %d", len(QuizTests), len(mgr.GetAll()))
	}
}

func TestConcurrent(t *testing.T) {
	mgr := NewManager()

	done := make(chan struct{}, len(QuizTests))
	for _, elem := range QuizTests {
		go func(e Quiz) {
			mgr.Load(e)
			done <- struct{}{}
		}(elem)
	}
	for i := 0; i < len(QuizTests); i++ {
		<-done
	}

	for _, elem := range QuizTests {
		go func(e Quiz) {
			ent := mgr.Get(e.Hash())
			if ent.String() != e.String() {
				t.Errorf("get returned incorrect value: got %v, expected %v", ent, e)
			}
			done <- struct{}{}
		}(elem)
	}
	for i := 0; i < len(QuizTests); i++ {
		<-done
	}
}
