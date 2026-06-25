package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	lowercase = "abcdefghijklmnopqrstuvwxyz"
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits    = "0123456789"
	symbols   = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

// Options configures password generation.
type Options struct {
	Length    int
	Uppercase bool
	Lowercase bool
	Digits    bool
	Symbols   bool
}

// DefaultOptions returns sensible defaults for password generation.
func DefaultOptions() Options {
	return Options{
		Length:    24,
		Uppercase: true,
		Lowercase: true,
		Digits:    true,
		Symbols:   true,
	}
}

// Generate creates a cryptographically random password with the given options.
func Generate(opts Options) (string, error) {
	if opts.Length <= 0 {
		return "", fmt.Errorf("password length must be positive, got %d", opts.Length)
	}

	var charset strings.Builder
	var required []byte

	if opts.Lowercase {
		charset.WriteString(lowercase)
		ch, err := randomChar(lowercase)
		if err != nil {
			return "", err
		}
		required = append(required, ch)
	}
	if opts.Uppercase {
		charset.WriteString(uppercase)
		ch, err := randomChar(uppercase)
		if err != nil {
			return "", err
		}
		required = append(required, ch)
	}
	if opts.Digits {
		charset.WriteString(digits)
		ch, err := randomChar(digits)
		if err != nil {
			return "", err
		}
		required = append(required, ch)
	}
	if opts.Symbols {
		charset.WriteString(symbols)
		ch, err := randomChar(symbols)
		if err != nil {
			return "", err
		}
		required = append(required, ch)
	}

	pool := charset.String()
	if len(pool) == 0 {
		return "", fmt.Errorf("at least one character set must be enabled")
	}

	if opts.Length < len(required) {
		return "", fmt.Errorf("password length %d is too short for required character sets", opts.Length)
	}

	// Generate remaining characters.
	result := make([]byte, opts.Length)
	copy(result, required)

	for i := len(required); i < opts.Length; i++ {
		ch, err := randomChar(pool)
		if err != nil {
			return "", err
		}
		result[i] = ch
	}

	// Shuffle the result using Fisher-Yates.
	for i := len(result) - 1; i > 0; i-- {
		j, err := randomInt(i + 1)
		if err != nil {
			return "", err
		}
		result[i], result[j] = result[j], result[i]
	}

	return string(result), nil
}

func randomChar(charset string) (byte, error) {
	idx, err := randomInt(len(charset))
	if err != nil {
		return 0, err
	}
	return charset[idx], nil
}

func randomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random int: %w", err)
	}
	return int(n.Int64()), nil
}
