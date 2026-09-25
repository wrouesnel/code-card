// Package codecard generates credit-card sized grids of random letter to digit associations
// used to recall PIN numbers from a secret keyword.
package codecard

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	mrand "math/rand/v2"
	"strings"

	"github.com/pkg/errors"
	"github.com/wrouesnel/code-card/pkg/pyrandom"
)

const (
	letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits  = "0123456789"

	// fullSamples is the number of times every digit is assigned to a letter before the
	// remaining letters are filled randomly, to ensure diversity and balance repeats.
	fullSamples = len(letters) / len(digits)

	// MinSaltLength is the minimum salt length in bytes for encoded tables.
	MinSaltLength = 16
	// MaxRepeats is the maximum number of times a letter may repeat in an encoding keyword.
	MaxRepeats = 3
)

var (
	ErrSaltTooShort    = errors.New("salt too short")
	ErrKeywordTooShort = errors.New("the keyword must be at least as long as the code")
	ErrTooManyRepeats  = errors.New("too many repeated characters in the keyword")
	ErrInvalidKeyword  = errors.New("the keyword may only contain the letters A-Z")
	ErrInvalidCode     = errors.New("the code may only contain the digits 0-9")
)

// Table maps each letter A-Z (by index) to one or more digits.
type Table [len(letters)][]byte

// Rows returns the table as display strings of the form "A 1 2", in letter order.
func (t *Table) Rows() []string {
	rows := make([]string, len(t))
	for idx, values := range t {
		parts := []string{string(letters[idx])}
		for _, value := range values {
			parts = append(parts, string(value))
		}
		rows[idx] = strings.Join(parts, " ")
	}
	return rows
}

// sourceTable returns the base letter to digit assignment: every digit is used fullSamples times
// in order, and the remaining letters are chosen with choice.
func sourceTable(choice func() byte) []byte {
	src := make([]byte, len(letters))
	for idx := range fullSamples * len(digits) {
		src[idx] = digits[idx%len(digits)]
	}
	for idx := fullSamples * len(digits); idx < len(letters); idx++ {
		src[idx] = choice()
	}
	return src
}

// cryptoSource is a math/rand/v2 source backed by crypto/rand.
type cryptoSource struct{}

func (cryptoSource) Uint64() uint64 {
	var buf [8]byte
	_, _ = rand.Read(buf[:]) // never returns an error
	return binary.LittleEndian.Uint64(buf[:])
}

// RandomTable generates a totally random table from the system's secure random source.
func RandomTable() Table {
	rng := mrand.New(cryptoSource{}) //nolint:gosec // backed by crypto/rand
	src := sourceTable(func() byte { return digits[rng.IntN(len(digits))] })
	rng.Shuffle(len(src), func(i, j int) { src[i], src[j] = src[j], src[i] })

	var table Table
	for idx, value := range src {
		table[idx] = []byte{value}
	}
	return table
}

// SemiRandomTable generates a table which encodes code against keyword: reading the digits
// of the keyword's letters from the card yields the code. When a letter repeats in the keyword,
// its first occurrence is read from the first column of that letter's digits, the second from
// the second column, and so on.
//
// Semi-random tables could leak information if multiple dis-similar cards were generated for the
// same code and keyword. So although randomized, the table is deterministic: the generator is
// seeded with a hash of the salt, code and keyword. Output is identical to the original Python
// implementation for the same inputs.
//
// This has a failure mode with bad keywords: e.g. AAAAAA for 123456 would put the whole PIN
// against the letter A.
func SemiRandomTable(salt, code, keyword string) (Table, error) {
	var table Table

	if len(salt) < MinSaltLength {
		return table, errors.Wrapf(ErrSaltTooShort, "please set a salt which is at least %d bytes long (yours: %d bytes)",
			MinSaltLength, len(salt))
	}
	if len(keyword) < len(code) {
		return table, ErrKeywordTooShort
	}
	if strings.Trim(code, digits) != "" {
		return table, ErrInvalidCode
	}
	upperKeyword := strings.ToUpper(keyword)
	if keyword == "" || strings.Trim(upperKeyword, letters) != "" {
		return table, ErrInvalidKeyword
	}

	counts := map[byte]int{}
	repeats := 0
	for idx := range len(upperKeyword) {
		counts[upperKeyword[idx]]++
		repeats = max(repeats, counts[upperKeyword[idx]])
	}
	// Keep repeats sensibly limited
	if repeats > MaxRepeats {
		return table, fmt.Errorf("%w: %d > %d", ErrTooManyRepeats, repeats, MaxRepeats)
	}

	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(code))
	h.Write([]byte(keyword))
	rng := pyrandom.New(h.Sum(nil))
	choice := func() byte { return pyrandom.Choice(rng, []byte(digits)) }

	// Build a table with random repeats in it, up to the maximum.
	for idx, value := range sourceTable(choice) {
		table[idx] = []byte{value}
		for range repeats - 1 {
			table[idx] = append(table[idx], choice())
		}
	}

	// Set the letters we must. The nth use of a letter in the keyword sets its nth position.
	// Since the table was built with the maximum number of repeats, every position exists.
	setCounts := map[byte]int{}
	for idx := range len(code) {
		letter := upperKeyword[idx]
		table[letter-'A'][setCounts[letter]] = code[idx]
		setCounts[letter]++
	}

	return table, nil
}
