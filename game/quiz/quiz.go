package quiz

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/url"
	"time"
)

// Quiz handling constants.
const (
	// MaxQuizSize is the 8 MB quiz size limit.
	MaxQuizSize = 8 * 1024 * 1024
)

// Quiz sources. Used for internal bookkeeping.
const (
	// Loaded from the filesystem at startup.
	SourceFilesystem = iota
	// Loaded from a server peer.
	SourceNetwork
	// Loaded from a user upload.
	SourceUpload
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
	Duration int      `json:"time"`
	ImageURL *url.URL `json:"image_url"`
	Answers  []Answer `json:"answers"`
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
	Category    string     `json:"category"`
	Created     time.Time  `json:"created"`
	Questions   []Question `json:"questions"`

	// Internal variables for bookkeeping
	hash     hash.Hash
	inserted time.Time
	source   int
}

// LoadQuiz buffers and parses a quiz archive file, returning the loaded
// object. Origin is the game source used.
func LoadQuiz(src io.Reader, origin int) (Quiz, error) {
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

	q.source = origin
	return q, nil
}

// Hash returns the unique hash associated with this game instance. Hashes for
// Gahoot game archives are performed based on JSON output including EVERY
// field in the struct, even if those were not present in the original version.
// This prevents the addition of blank fields to a JSON document from
// completely changing the hash.
func (q *Quiz) Hash() hash.Hash {
	if q.hash == nil {
		// NOTE: Deliberately not error checking here, as it is
		// unlikely we will get a result insufficient for hashing
		buf, _ := json.Marshal(q)
		h := sha256.New()
		h.Write(buf)

		q.hash = h
		return h
	}

	return q.hash
}

// Remote returns if this quiz was obtained from a remote source.
func (q Quiz) Remote() bool {
	return q.source != SourceFilesystem
}

// Category returns category if it is non-empty, else "Uncategorised".
func (q Quiz) FriendlyCategory() string {
	if q.Category == "" {
		return "Uncategorised"
	}

	return q.Category
}

// String returns a stringified representation of this Quiz's unique hash.
func (q Quiz) String() string {
	h := q.Hash()
	return fmt.Sprintf("%X", h.Sum(nil))
}
