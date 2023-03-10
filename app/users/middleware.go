package users

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"

	"github.com/badu/microservices-demo/pkg/config"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type SessionManager struct {
	logger  logger.Logger
	cfg     *config.Config
	service Service
}

func NewSessionManager(logger logger.Logger, cfg *config.Config, service Service) SessionManager {
	return SessionManager{logger: logger, cfg: cfg, service: service}
}

type RequestCtxUser struct{}

type RequestCtxSession struct{}

func (m *SessionManager) SessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "users_middlewares.SessionMiddleware")
		defer span.Finish()

		cookie, err := c.Cookie(m.cfg.HttpServer.SessionCookieName)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				m.logger.Errorf("SessionMiddleware.ErrNoCookie: %v", err)
				return httpErrors.ErrorCtxResponse(c, err)
			}
			m.logger.Errorf("SessionMiddleware.c.Cookie: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		sessionByID, err := m.service.GetSessionByID(ctx, cookie.Value)
		if err != nil {
			m.logger.Errorf("SessionMiddleware.GetSessionByID: %v", err)
			return httpErrors.ErrorCtxResponse(c, httpErrors.Unauthorized)
		}

		userResponse, err := m.service.GetByID(ctx, sessionByID.UserID)
		if err != nil {
			m.logger.Errorf("SessionMiddleware.service.GetByID: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		ctx = context.WithValue(c.Request().Context(), RequestCtxUser{}, userResponse)
		ctx = context.WithValue(ctx, RequestCtxSession{}, sessionByID)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}
