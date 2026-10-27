package authn

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

const argonPrefix = "argon2id"

type Argon2id struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

func DefaultArgon2id() Argon2id {
	return Argon2id{Memory: 64 * 1024, Time: 3, Threads: 2, KeyLen: 32, SaltLen: 16}
}

func FastArgon2id() Argon2id {
	return Argon2id{Memory: 8 * 1024, Time: 1, Threads: 1, KeyLen: 32, SaltLen: 16}
}

func (a Argon2id) Hash(password string) (string, error) {
	salt := make([]byte, a.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2: salt: %w", err)
	}
	sum := argon2.IDKey([]byte(password), salt, a.Time, a.Memory, a.Threads, a.KeyLen)
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argonPrefix,
		argon2.Version,
		a.Memory, a.Time, a.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	), nil
}

func (a Argon2id) Verify(encoded, password string) error {
	memory, timeCost, threads, salt, want, err := parseArgon(encoded)
	if err != nil {
		return errx.ErrUnauthenticated
	}
	got := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(want, got) != 1 {
		return errx.ErrUnauthenticated
	}
	return nil
}

func parseArgon(encoded string) (memory uint32, timeCost uint32, threads uint8, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != argonPrefix {
		return 0, 0, 0, nil, nil, fmt.Errorf("argon2: malformed hash")
	}
	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	if hash, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	return memory, timeCost, threads, salt, hash, nil
}
