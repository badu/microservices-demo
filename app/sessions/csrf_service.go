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

type CSRFRepository interface {
	Create(ctx context.Context, token string) error
	GetToken(ctx context.Context, token string) (string, error)
}

type CsrfServiceImpl struct {
	csrfRepo CSRFRepository
}

func NewCSRFService(csrfRepo CSRFRepository) CsrfServiceImpl {
	return CsrfServiceImpl{csrfRepo: csrfRepo}
}

func (c *CsrfServiceImpl) GetCSRFToken(ctx context.Context, sesID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CsrfServiceImpl.CreateToken")
	defer span.Finish()

	token, err := c.makeToken(sesID)
	if err != nil {
		return "", errors.Wrap(err, "CsrfServiceImpl.CreateToken.c.makeToken")
	}

	if err := c.csrfRepo.Create(ctx, token); err != nil {
		return "", errors.Wrap(err, "CsrfServiceImpl.CreateToken.csrfRepo.Create")
	}

	return token, nil
}

func (c *CsrfServiceImpl) ValidateCSRFToken(ctx context.Context, sesID string, token string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CsrfServiceImpl.CheckToken")
	defer span.Finish()

	existsToken, err := c.csrfRepo.GetToken(ctx, token)
	if err != nil {
		return false, err
	}

	return c.validateToken(existsToken, sesID), nil
}

func (c *CsrfServiceImpl) makeToken(sessionID string) (string, error) {
	hash := sha256.New()
	_, err := io.WriteString(hash, csrfSalt+sessionID)
	if err != nil {
		return "", err
	}
	token := base64.RawStdEncoding.EncodeToString(hash.Sum(nil))
	return token, nil
}

func (c *CsrfServiceImpl) validateToken(token string, sessionID string) bool {
	trueToken, err := c.makeToken(sessionID)
	if err != nil {
		return false
	}
	return token == trueToken
}
