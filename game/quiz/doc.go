// Package quiz implements Gahoot quiz archive serialisation and parsing, as
// well as hashing and integrity checking.
//
// Every quiz in Gahoot is uniquely identified by its SHA256 hash, which is
// calculated based on the following simple algorithm:
//	1) Re-order fields to be in the same order as quiz.Quiz
//	2) Add any missing fields in the same corresponding position with their
//	   Go zero value
//	3) Minify such that any non-significant whitespace is elided
//	4) Take a SHA256 hash over the resulting raw bytes
//
// This hashing system and protocol allows a way for gameservers to exchange
// games if needed, and for individual servers to easily keep track of what
// games are uploaded, without needing to do searches for game titles, etc.
// This also means that different versions of a game can have the same title
// and still be programmatically distinct. A combination of identifying
// information is, therefore, included in the format to aid in differentiating
// quizzes.
//
// Package quiz also implements a quiz manager, which simply manages a central
// store of quizzes which can be re-used, updated and cleaned on request. This
// is practically a glorified map and mutex, but it is somewhat unexpectedly
// intricate, as quizzes are stored based on the textual representation of
// their hash rather than the actual object itself. This is due to unforeseen
// issues with copying hash.Hash objects, causing their internal state to
// change and their hash to be altered. This has the added benefit of allowing
// user searches of the database to be trivial.
package quiz
