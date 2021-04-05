package sessions

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

type CSRFService interface {
	GetCSRFToken(ctx context.Context, sesID string) (string, error)
	ValidateCSRFToken(ctx context.Context, sesID string, token string) (bool, error)
}

const (
	CSRFHeader = "X-CSRF-Token"
	// 32 bytes
	csrfSalt = "KbWaoi5xtDC3GEfBa9ovQdzOzXsuVU9I"
)

type csrfServiceImpl struct {
	csrfRepo CSRFRepository
}

func NewCSRFService(csrfRepo CSRFRepository) *csrfServiceImpl {
	return &csrfServiceImpl{csrfRepo: csrfRepo}
}

func (c *csrfServiceImpl) GetCSRFToken(ctx context.Context, sesID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrfServiceImpl.CreateToken")
	defer span.Finish()

	token, err := c.makeToken(sesID)
	if err != nil {
		return "", errors.Wrap(err, "csrfServiceImpl.CreateToken.c.makeToken")
	}

	if err := c.csrfRepo.Create(ctx, token); err != nil {
		return "", errors.Wrap(err, "csrfServiceImpl.CreateToken.csrfRepo.Create")
	}

	return token, nil
}

func (c *csrfServiceImpl) ValidateCSRFToken(ctx context.Context, sesID string, token string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrfServiceImpl.CheckToken")
	defer span.Finish()

	existsToken, err := c.csrfRepo.GetToken(ctx, token)
	if err != nil {
		return false, err
	}

	return c.validateToken(existsToken, sesID), nil
}

func (c *csrfServiceImpl) makeToken(sessionID string) (string, error) {
	hash := sha256.New()
	_, err := io.WriteString(hash, csrfSalt+sessionID)
	if err != nil {
		return "", err
	}
	token := base64.RawStdEncoding.EncodeToString(hash.Sum(nil))
	return token, nil
}

func (c *csrfServiceImpl) validateToken(token string, sessionID string) bool {
	trueToken, err := c.makeToken(sessionID)
	if err != nil {
		return false
	}
	return token == trueToken
}
