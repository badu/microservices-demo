package comments

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/gateway/users"
	"github.com/badu/microservices-demo/pkg/config"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	CreateComment(ctx context.Context, comment *Comment) (*Comment, error)
	GetCommByID(ctx context.Context, commentID uuid.UUID) (*Comment, error)
	UpdateComment(ctx context.Context, comment *Comment) (*Comment, error)
	GetByHotelID(ctx context.Context, hotelID uuid.UUID, page, size int64) (*List, error)
}

type ServerImpl struct {
	cfg      *config.Config
	group    *echo.Group
	logger   logger.Logger
	validate *validator.Validate
	service  Service
}

func NewServer(
	cfg *config.Config,
	group *echo.Group,
	logger logger.Logger,
	validate *validator.Validate,
	service Service,
) ServerImpl {
	return ServerImpl{cfg: cfg, group: group, logger: logger, validate: validate, service: service}
}

// Register CreateComment
// @Tags Comments
// @Summary Create new comment
// @Description Create new single comment
// @Accept json
// @Produce json
// @Success 201 {object} Comment
// @Router /comments [post]
// @BasePath /api/v1
func (s *ServerImpl) CreateComment() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "ServerImpl.CreateComment")
		defer span.Finish()

		var comm Comment
		if err := c.Bind(&comm); err != nil {
			s.logger.Error("c.Bind")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		if err := s.validate.StructCtx(ctx, &comm); err != nil {
			s.logger.Error("validate.StructCtx")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		comment, err := s.service.CreateComment(ctx, &comm)
		if err != nil {
			s.logger.Error("service.CreateComment")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusCreated, comment)
	}
}

// Register GetCommByID
// @Tags Comments
// @Summary Get comment by id
// @Description Get comment by uuid
// @Accept json
// @Produce json
// @Param comment_id query string false "comment uuid"
// @Success 200 {object} Comment
// @Router /comments/{comment_id} [get]
// @BasePath /api/v1
func (s *ServerImpl) GetCommByID() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "ServerImpl.GetCommByID")
		defer span.Finish()

		commUUID, err := uuid.FromString(c.QueryParam("comment_id"))
		if err != nil {
			s.logger.Error("uuid.FromString")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		commByID, err := s.service.GetCommByID(ctx, commUUID)
		if err != nil {
			s.logger.Error("service.GetCommByID")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, commByID)
	}
}

// Register UpdateComment
// @Tags Comments
// @Summary Update comment by id
// @Description Update comment by uuid
// @Accept json
// @Produce json
// @Param comment_id query string false "comment uuid"
// @Success 200 {object} Comment
// @Router /comments/{comment_id} [put]
// @BasePath /api/v1
func (s *ServerImpl) UpdateComment() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "ServerImpl.UpdateComment")
		defer span.Finish()

		commUUID, err := uuid.FromString(c.QueryParam("comment_id"))
		if err != nil {
			s.logger.Error("uuid.FromString")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		var comm Comment
		if err := c.Bind(&comm); err != nil {
			s.logger.Error("c.Bind")
			return httpErrors.ErrorCtxResponse(c, err)
		}
		comm.CommentID = commUUID

		if err := s.validate.StructCtx(ctx, &comm); err != nil {
			s.logger.Error("validate.StructCtx")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		comment, err := s.service.UpdateComment(ctx, &comm)
		if err != nil {
			s.logger.Error("service.UpdateComment")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, comment)
	}
}

// Register GetByHotelID
// @Tags Comments
// @Summary Get comments by hotel id
// @Description Get comments list by hotel uuid
// @Accept json
// @Produce json
// @Param hotel_id query string false "hotel uuid"
// @Success 200 {object} List
// @Router /comments/hotel/{hotel_id} [get]
// @BasePath /api/v1
func (s *ServerImpl) GetByHotelID() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "ServerImpl.GetByHotelID")
		defer span.Finish()

		hotelUUID, err := uuid.FromString(c.QueryParam("hotel_id"))
		if err != nil {
			s.logger.Error("uuid.FromString")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		page, err := strconv.Atoi(c.QueryParam("page"))
		if err != nil {
			s.logger.Error("strconv.Atoi")
			return httpErrors.ErrorCtxResponse(c, err)
		}
		size, err := strconv.Atoi(c.QueryParam("size"))
		if err != nil {
			s.logger.Error("strconv.Atoi")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		commentsList, err := s.service.GetByHotelID(ctx, hotelUUID, int64(page), int64(size))
		if err != nil {
			s.logger.Error("service.GetByHotelID")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, commentsList)
	}
}

func (s *ServerImpl) MapRoutes(mw *users.SessionMiddleware) {
	s.group.GET("/:comment_id", s.GetCommByID())
	s.group.POST("", s.CreateComment(), mw.SessionMiddleware)
	s.group.PUT("/:comment_id", s.UpdateComment(), mw.SessionMiddleware)
	s.group.PUT("/comments/hotel/:hotel_id", s.GetByHotelID())
}
