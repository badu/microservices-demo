package sessions

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"github.com/opentracing/opentracing-go"
)

type CSRFService interface {
	GetCSRFToken(ctx context.Context, sesID string) (string, error)
	ValidateCSRFToken(ctx context.Context, sesID string, token string) (bool, error)
}

type CSRFRepository interface {
	Create(ctx context.Context, token string) error
	GetToken(ctx context.Context, token string) (string, error)
}

type CsrfServiceImpl struct {
	repository CSRFRepository
	salt       string
}

func NewCSRFService(salt string, csrfRepo CSRFRepository) CsrfServiceImpl {
	return CsrfServiceImpl{repository: csrfRepo, salt: salt}
}

func (c *CsrfServiceImpl) GetCSRFToken(ctx context.Context, sesID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_service.CreateToken")
	defer span.Finish()

	token, err := c.makeToken(sesID)
	if err != nil {
		return "", errors.Join(err, errors.New("CsrfServiceImpl.CreateToken.c.makeToken"))
	}

	if err := c.repository.Create(ctx, token); err != nil {
		return "", errors.Join(err, errors.New("CsrfServiceImpl.CreateToken.repository.Create"))
	}

	return token, nil
}

func (c *CsrfServiceImpl) ValidateCSRFToken(ctx context.Context, sesID string, token string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_service.CheckToken")
	defer span.Finish()

	existsToken, err := c.repository.GetToken(ctx, token)
	if err != nil {
		return false, err
	}

	return c.validateToken(existsToken, sesID), nil
}

func (c *CsrfServiceImpl) makeToken(sessionID string) (string, error) {

	hash := sha256.New()
	_, err := io.WriteString(hash, c.salt+sessionID)
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
