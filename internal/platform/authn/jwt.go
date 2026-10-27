package authn

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const issuer = "treatment-plus"

type Claims struct {
	ClinicID string `json:"clinic_id"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret []byte
	ttl    time.Duration
	clock  clock.Clock
}

func NewJWT(cfg config.Auth, clk clock.Clock) (*JWT, error) {
	if len(cfg.JWTAccessSecret) < 32 {
		return nil, fmt.Errorf("authn: JWT_ACCESS_SECRET must be at least 32 bytes")
	}
	if clk == nil {
		clk = clock.System{}
	}
	ttl := cfg.AccessTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &JWT{secret: []byte(cfg.JWTAccessSecret), ttl: ttl, clock: clk}, nil
}

func (j *JWT) Issue(clinicID id.ID, now time.Time) (string, time.Duration, error) {
	if now.IsZero() {
		now = j.clock.Now()
	}
	claims := Claims{
		ClinicID: clinicID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   clinicID.String(),
			ID:        id.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return "", 0, fmt.Errorf("authn: sign: %w", err)
	}
	return signed, j.ttl, nil
}

func (j *JWT) ParseAccess(raw string) (id.ID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected method")
		}
		return j.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithTimeFunc(j.clock.Now), jwt.WithLeeway(5*time.Second))
	if err != nil || !token.Valid {
		return id.ID{}, errx.ErrUnauthenticated
	}
	if claims.ClinicID == "" {
		claims.ClinicID = claims.Subject
	}
	parsed, err := id.Parse(claims.ClinicID)
	if err != nil {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return parsed, nil
}

func Bearer(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
