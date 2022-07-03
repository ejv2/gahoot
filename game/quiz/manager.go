package quiz

import (
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

// LoadDirError is the error returned for non-fatal errors during crawling a
// directory. To prevent errors in a single file prevent all other files being
// loaded, the process is not stopped unless an error which makes it impossible
// to continue is encountered. Instead, errors are added to this error, which
// is an alias for a slice of errors.
type LoadDirError []error

func (e LoadDirError) Error() string {
	sb := &strings.Builder{}
	for _, elem := range e {
		sb.WriteString(elem.Error() + "\n")
	}

	return sb.String()
}

// Manager is the quiz manager, responsible for memory caching and loading new
// quizzes into memory, as well as handling periodic cleans of the cache, based
// on specified memory timeouts.
type Manager struct {
	mut *sync.RWMutex
	// qs maps a stringified hash value to a quiz
	qs map[string]Quiz
	// cats contains registered categories
	cats map[string]bool
}

// NewManager allocates and returns a GameManager ready for use.
func NewManager() Manager {
	return Manager{
		mut:  new(sync.RWMutex),
		qs:   make(map[string]Quiz),
		cats: make(map[string]bool),
	}
}

// Load attempts to store the passed quiz into the game map. If the entry is
// already present or if the maximum entries are already present, error is
// non-nil.
func (m *Manager) Load(q Quiz) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	return m.load(q)
}

// load is a non-synchronised version of Load which assumes that m.mut is held.
func (m *Manager) load(q Quiz) error {
	h := q.String()
	if _, ok := m.qs[h]; ok {
		return fmt.Errorf("quizman: load: duplicate entry")
	}
	q.inserted = time.Now()

	m.cats[q.Category] = true

	m.qs[h] = q
	return nil
}

// LoadFrom loads and parses a quiz archive file and attempts to add into the
// manager store. If parsing failed, an empty quiz is returned and the
// resulting error. If loading into the store failed, the parsed quiz is
// returned with an error.
func (m *Manager) LoadFrom(path string) (Quiz, error) {
	f, err := os.Open(path)
	if err != nil {
		return Quiz{}, fmt.Errorf("quizman: loadfrom: %w", err)
	}

	q, err := LoadQuiz(f, SourceFilesystem)
	if err != nil {
		return Quiz{}, err
	}

	err = m.Load(q)
	if err != nil {
		return q, err
	}

	return q, nil
}

// LoadDir recursively loads all quiz archives from path and any descendent
// directories. If no errors are encountered, error is nil. If any non-fatal
// errors were encountered, error is LoadDirError, which is a slice of other
// errors. Fatal errors are returned as-is.
func (m *Manager) LoadDir(path string) ([]Quiz, error) {
	ent, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	m.mut.Lock()
	defer m.mut.Unlock()

	qs := make([]Quiz, 0, len(m.qs)+len(ent))
	errs := make(LoadDirError, 0)
	for _, elem := range ent {
		full := filepath.Join(path, elem.Name())

		if elem.IsDir() {
			m.mut.Unlock()
			sub, err := m.LoadDir(full)
			m.mut.Lock()
			if err != nil {
				errs = append(errs, err)
				continue
			}

			qs = append(qs, sub...)
			continue
		}

		f, err := os.Open(full)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		q, err := LoadQuiz(f, SourceFilesystem)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = m.load(q)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		qs = append(qs, q)
	}

	if len(errs) == 0 {
		return qs, nil
	}
	return qs, errs
}

// Get fetches a quiz with the corresponding hash.
func (m *Manager) Get(h hash.Hash) (Quiz, bool) {
	m.mut.RLock()
	defer m.mut.RUnlock()

	q, ok := m.qs[fmt.Sprintf("%X", h.Sum(nil))]
	return q, ok
}

// GetString fetches a quiz with the corresponding stringfied hash.
func (m *Manager) GetString(h string) (Quiz, bool) {
	m.mut.Lock()
	defer m.mut.RUnlock()

	q, ok := m.qs[h]
	return q, ok
}

// GetAll returns every registered quiz from the quiz map.
func (m *Manager) GetAll() []Quiz {
	m.mut.RLock()
	defer m.mut.RUnlock()

	all := make([]Quiz, 0, len(m.qs))
	for _, quiz := range m.qs {
		all = append(all, quiz)
	}

	return all
}

// GetCategories returns all recognised distinct categories from loaded game
// archives. The returned slice is guaranteed to contain no duplicates or
// unused categories.
func (m *Manager) GetCategories() []string {
	m.mut.RLock()
	defer m.mut.RUnlock()

	cats := make([]string, 0, len(m.cats))
	for cat := range m.cats {
		if cat == "" {
			cat = "Uncategorised"
		}

		buf := []rune(cat)
		buf[0] = unicode.ToTitle(buf[0])

		cats = append(cats, string(buf))
	}

	return cats
}
