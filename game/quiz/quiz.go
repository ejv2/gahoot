package quiz

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/url"
	"time"
)

// Quiz handling constants
const (
	// MaxQuizSize is the 8 MB quiz size limit
	MaxQuizSize = 8 * 1024 * 1024
)

// An Answer is one option in a single question in a quiz.
// It is simply a response title and a boolean for if the response is
// acceptable as correct.
type Answer struct {
	Title   string `json:"title"`
	Correct bool   `json:"correct"`
}

// A Question represents a single question in a quiz.
// It consists of a question statement ("What is a cat?") and a set of responses
// ("Mamal", "Bird", "Plane","Superman"). Correct is the index of the correct
// response.
type Question struct {
	Title    string   `json:"title"`
	ImageURL url.URL  `json:"image_url"`
	Answers  []Answer `json:"answer"`
}

// A Quiz represents a quiz which can be played in Gahoot. It is indirectly
// represented through a Gahoot game archive, which can be uploaded and played
// onto any Gahoot instance.
//
// A Gahoot game archive is defined as the resulting JSON from calling the Go
// standard library's encoding/json.Marshall function upon the Quiz struct,
// thereby encoding all child structs. This resulting single-line JSON text is
// the source used to calculate hash, which uniquely identifies a game across
// all Gahoot servers.
type Quiz struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	Created     time.Time  `json:"created"`
	Questions   []Question `json:"questions"`

	// hash is only used when hash is pre-calculated
	hash hash.Hash
}

func LoadQuiz(src io.Reader) (Quiz, error) {
	r := io.LimitReader(src, MaxQuizSize)
	buf, err := io.ReadAll(r)

	if err != nil && err != io.EOF {
		return Quiz{}, fmt.Errorf("quiz: load: %w", err)
	}
	if len(buf) == 0 {
		return Quiz{}, fmt.Errorf("quiz: load: empty")
	}

	// One-byte over-read to check for quiz truncation and report an
	// accurate error
	var tmp [1]byte
	n, err := src.Read(tmp[:])
	if n != 0 && err != io.EOF {
		return Quiz{}, fmt.Errorf("quiz: load: too large")
	}

	var q Quiz
	err = json.Unmarshal(buf, &q)
	if err != nil {
		return Quiz{}, fmt.Errorf("quiz: load: %w", err)
	}

	// As we have the source upfront, we can just calculate it now
	// and cache value for later.
	// NOTE: When this is done, we *cannot* modify quiz in memory
	compact := bytes.NewBuffer(buf)
	json.Compact(compact, buf)
	q.hash = sha256.New()
	q.hash.Write(compact.Bytes())

	return q, nil
}

func (q Quiz) Hash() hash.Hash {
	if q.hash != nil {
		// NOTE: Deliberately not error checking here, as it is
		// unlikely we will get a result insufficient for hashing
		buf, _ := json.Marshal(q)
		h := sha256.New()
		h.Write(buf)

		return h
	}

	return q.hash
}