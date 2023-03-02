package users

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/images"
	"github.com/badu/microservices-demo/app/sessions"

	"github.com/badu/microservices-demo/pkg/config"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

const (
	csrfHeader  = "X-CSRF-Token"
	maxFileSize = 1024 * 1024 * 10
)

type HTTPTransport struct {
	cfg      *config.Config
	group    *echo.Group
	service  Service
	logger   logger.Logger
	validate *validator.Validate
	mw       *MiddlewareManager
}

func NewHTTPServer(
	group *echo.Group,
	service Service,
	logger logger.Logger,
	validate *validator.Validate,
	cfg *config.Config,
	mw *MiddlewareManager,
) HTTPTransport {
	return HTTPTransport{group: group, service: service, logger: logger, validate: validate, cfg: cfg, mw: mw}
}

// Register godoc
// @Summary Register new user
// @Tags UserDO
// @Description register new user account, returns user data and session
// @Accept json
// @Produce json
// @Param data body UserDO true "user data"
// @Success 201 {object} UserResponse
// @Router /user/register [post]
// @BasePath /api/v1
func (h *HTTPTransport) Register() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "auth.Register")
		defer span.Finish()

		var u UserDO
		if err := c.Bind(&u); err != nil {
			h.logger.Errorf("c.Bind: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		if err := h.validate.StructCtx(ctx, &u); err != nil {
			h.logger.Errorf("validate.StructCtx: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		regUser, err := h.service.Register(ctx, &u)
		if err != nil {
			h.logger.Errorf("HTTPTransport.Register.service.Register: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		sessionID, err := h.service.CreateSession(ctx, regUser.UserID)
		if err != nil {
			h.logger.Errorf("HTTPTransport.service.CreateSession: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		c.SetCookie(&http.Cookie{
			Name:     h.cfg.HttpServer.SessionCookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(time.Duration(h.cfg.HttpServer.CookieLifeTime) * time.Minute),
		})

		return c.JSON(http.StatusCreated, regUser)
	}
}

// Login godoc
// @Summary Login user
// @Tags UserDO
// @Description login user, returns user data and session
// @Accept json
// @Produce json
// @Param data body Login true "email and password"
// @Success 200 {object} UserResponse
// @Router /user/login [post]
// @BasePath /api/v1
func (h *HTTPTransport) Login() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.Login")
		defer span.Finish()

		var login Login
		if err := c.Bind(&login); err != nil {
			h.logger.Errorf("c.Bind: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		if err := h.validate.StructCtx(ctx, &login); err != nil {
			h.logger.Errorf("validate.StructCtx: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		userResponse, err := h.service.Login(ctx, login)
		if err != nil {
			h.logger.Errorf("HTTPTransport.service.Login: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		sessionID, err := h.service.CreateSession(ctx, userResponse.UserID)
		if err != nil {
			h.logger.Errorf("HTTPTransport.Login.CreateSession: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		c.SetCookie(&http.Cookie{
			Name:     h.cfg.HttpServer.SessionCookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(time.Duration(h.cfg.HttpServer.CookieLifeTime) * time.Minute),
		})

		return c.JSON(http.StatusOK, userResponse)
	}
}

// Logout godoc
// @Summary Logout user
// @Tags UserDO
// @Description Logout user, return no content
// @Accept json
// @Produce json
// @Success 204 ""
// @Router /user/logout [post]
// @BasePath /api/v1
func (h *HTTPTransport) Logout() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.Logout")
		defer span.Finish()

		cookie, err := c.Cookie(h.cfg.HttpServer.SessionCookieName)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				h.logger.Errorf("HTTPTransport.Logout.http.ErrNoCookie: %v", err)
				return httpErrors.ErrorCtxResponse(c, err)
			}
			h.logger.Errorf("HTTPTransport.Logout.c.Cookie: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		if err := h.service.DeleteSession(ctx, cookie.Value); err != nil {
			h.logger.Errorf("HTTPTransport.service.DeleteSession: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		c.SetCookie(&http.Cookie{
			Name:   h.cfg.HttpServer.SessionCookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

		return c.NoContent(http.StatusNoContent)
	}
}

// GetMe godoc
// @Summary Get current user data
// @Tags UserDO
// @Description Get current user data, required session cookie
// @Accept json
// @Produce json
// @Success 200 {object} UserResponse
// @Router /user/me [get]
// @BasePath /api/v1
func (h *HTTPTransport) GetMe() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.GetMe")
		defer span.Finish()

		userResponse, ok := ctx.Value(RequestCtxUser{}).(*UserResponse)
		if !ok {
			h.logger.Error("invalid middleware user ctx")
			return httpErrors.ErrorCtxResponse(c, httpErrors.WrongCredentials)
		}

		return c.JSON(http.StatusOK, userResponse)
	}
}

// GetCSRFToken godoc
// @Summary Get csrf token
// @Tags UserDO
// @Description Get csrf token, required session
// @Accept json
// @Produce json
// @Success 204 ""
// @Router /user/csrf [get]
// @BasePath /api/v1
func (h *HTTPTransport) GetCSRFToken() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.GetCSRFToken")
		defer span.Finish()

		userSession, ok := ctx.Value(RequestCtxSession{}).(*sessions.SessionDO)
		if !ok {
			h.logger.Error("invalid middleware session ctx")
			return httpErrors.ErrorCtxResponse(c, httpErrors.WrongCredentials)
		}

		csrfToken, err := h.service.GetCSRFToken(ctx, userSession.SessionID)
		if err != nil {
			h.logger.Error("service.GetCSRFToken")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		c.Response().Header().Set(csrfHeader, csrfToken)

		return c.NoContent(http.StatusOK)
	}
}

// Update godoc
// @Summary Update user
// @Tags UserDO
// @Description update user profile
// @Accept json
// @Produce json
// @Success 200 {object} UserResponse
// @Router /user/{id} [get]
// @BasePath /api/v1
func (h *HTTPTransport) Update() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.Update")
		defer span.Finish()

		userID := c.Param("id")
		if userID == "" {
			h.logger.Error("invalid user id param")
			return httpErrors.ErrorCtxResponse(c, httpErrors.BadRequest)
		}

		userUUID, err := uuid.FromString(userID)
		if err != nil {
			h.logger.Error("invalid user uuid")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		var updUser UserUpdate
		if err := c.Bind(&updUser); err != nil {
			h.logger.Errorf("c.Bind: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}
		updUser.UserID = userUUID

		if err := h.validate.StructCtx(ctx, updUser); err != nil {
			h.logger.Errorf("HTTPTransport.validate.StructCtx: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		userResponse, err := h.service.Update(ctx, &updUser)
		if err != nil {
			h.logger.Errorf("HTTPTransport.service.Update: %v", err)
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, userResponse)
	}
}

func (h *HTTPTransport) Delete() echo.HandlerFunc {
	panic("implement me")
}

// GetUserByID godoc
// @Summary Get user by id
// @Tags UserDO
// @Description Get user data by id
// @Accept json
// @Produce json
// @Param id path int false "user uuid"
// @Success 200 {object} UserResponse
// @Router /user/{id} [get]
// @BasePath /api/v1
func (h *HTTPTransport) GetUserByID() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.GetUserByID")
		defer span.Finish()

		userID := c.Param("id")
		if userID == "" {
			h.logger.Error("invalid user id param")
			return httpErrors.ErrorCtxResponse(c, httpErrors.BadRequest)
		}

		userUUID, err := uuid.FromString(userID)
		if err != nil {
			h.logger.Error("uuid.FromString")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		userResponse, err := h.service.GetByID(ctx, userUUID)
		if err != nil {
			h.logger.Error("service.GetByID")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, userResponse)
	}
}

// UpdateAvatar godoc
// @Summary Update user avatar
// @Tags UserDO
// @Description Upload user avatar image
// @Accept mpfd
// @Produce json
// @Param id path int false "user uuid"
// @Success 200 {object} UserResponse
// @Router /user/{id}/avatar [put]
// @BasePath /api/v1
func (h *HTTPTransport) UpdateAvatar() echo.HandlerFunc {
	bufferPool := &sync.Pool{New: func() interface{} {
		return &bytes.Buffer{}
	}}
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "user.UpdateAvatar")
		defer span.Finish()

		userResponse, ok := ctx.Value(RequestCtxUser{}).(*UserResponse)
		if !ok {
			h.logger.Error("invalid middleware user ctx")
			return httpErrors.ErrorCtxResponse(c, httpErrors.WrongCredentials)
		}

		if err := c.Request().ParseMultipartForm(maxFileSize); err != nil {
			h.logger.Error("c.ParseMultipartForm")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxFileSize)
		defer c.Request().Body.Close()

		formFile, _, err := c.Request().FormFile("avatar")
		if err != nil {
			h.logger.Error("c.FormFile")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		fileType, err := h.checkAvatar(formFile)
		if err != nil {
			h.logger.Error("h.checkAvatar")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		buf, ok := bufferPool.Get().(*bytes.Buffer)
		if !ok {
			h.logger.Error("bufferPool.Get")
			return httpErrors.ErrorCtxResponse(c, httpErrors.InternalServerError)
		}
		defer bufferPool.Put(buf)
		buf.Reset()

		if _, err := io.Copy(buf, formFile); err != nil {
			h.logger.Error("io.Copy")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		data := &images.UpdateAvatarMsg{
			UserID:      userResponse.UserID,
			ContentType: fileType,
			Body:        buf.Bytes(),
		}

		if err := h.service.UpdateAvatar(ctx, data); err != nil {
			h.logger.Error("h.service.UpdateAvatar")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.NoContent(http.StatusOK)
	}
}

func (h *HTTPTransport) checkAvatar(file multipart.File) (string, error) {
	fileHeader := make([]byte, maxFileSize)
	ContentType := ""
	if _, err := file.Read(fileHeader); err != nil {
		return ContentType, err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return ContentType, err
	}

	count, err := file.Seek(0, 2)
	if err != nil {
		return ContentType, err
	}
	if count > maxFileSize {
		return ContentType, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return ContentType, err
	}
	ContentType = http.DetectContentType(fileHeader)

	if ContentType != "image/jpg" && ContentType != "image/png" && ContentType != "image/jpeg" {
		return ContentType, err
	}

	return ContentType, nil
}

func (h *HTTPTransport) MapUserRoutes() {
	h.group.POST("/register", h.Register())
	h.group.POST("/login", h.Login())
	h.group.PUT("/:id/avatar", h.UpdateAvatar(), h.mw.SessionMiddleware)
	h.group.GET("/:id", h.GetUserByID())
	h.group.PUT("/:id", h.Update(), h.mw.SessionMiddleware)
	h.group.GET("/me", h.GetMe(), h.mw.SessionMiddleware)
	h.group.GET("/csrf", h.GetCSRFToken(), h.mw.SessionMiddleware)
}
