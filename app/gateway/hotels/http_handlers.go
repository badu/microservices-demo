package hotels

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/gateway/middlewares"
	"github.com/badu/microservices-demo/pkg/config"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Server interface {
	CreateHotel() echo.HandlerFunc
	UpdateHotel() echo.HandlerFunc
	GetHotelByID() echo.HandlerFunc
	GetHotels() echo.HandlerFunc
	UploadImage() echo.HandlerFunc
}

type serverImpl struct {
	cfg      *config.Config
	group    *echo.Group
	logger   logger.Logger
	validate *validator.Validate
	service  Service
	mw       *middlewares.MiddlewareManager
}

func NewServer(
	cfg *config.Config,
	group *echo.Group,
	logger logger.Logger,
	validate *validator.Validate,
	service Service,
	mw *middlewares.MiddlewareManager,
) *serverImpl {
	return &serverImpl{cfg: cfg, group: group, logger: logger, validate: validate, service: service, mw: mw}
}

// Register CreateHotel
// @Tags Hotels
// @Summary Create new hotel
// @Description Create new hotel instance
// @Accept json
// @Produce json
// @Success 201 {object} Hotel
// @Router /hotels [post]
// @BasePath /api/v1
func (h *serverImpl) CreateHotel() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "serverImpl.CreateHotel")
		defer span.Finish()

		var hotelReq Hotel
		if err := c.Bind(&hotelReq); err != nil {
			h.logger.Error("c.Bind")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		if err := h.validate.StructCtx(ctx, &hotelReq); err != nil {
			h.logger.Error("validate.StructCtx")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		hotel, err := h.service.CreateHotel(ctx, &hotelReq)
		if err != nil {
			h.logger.Error("service.CreateHotel")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusCreated, hotel)
	}
}

// Register UpdateHotel
// @Tags Hotels
// @Summary Update hotel data
// @Description Update single hotel data
// @Accept json
// @Produce json
// @Param hotel_id path int true "Hotel UUID"
// @Success 200 {object} Hotel
// @Router /hotels/{hotel_id} [put]
// @BasePath /api/v1
func (h *serverImpl) UpdateHotel() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "serverImpl.UpdateHotel")
		defer span.Finish()

		hotelUUID, err := uuid.FromString(c.QueryParam("hotel_id"))
		if err != nil {
			h.logger.Error("uuid.FromString")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		var hotelReq Hotel
		if err := c.Bind(&hotelReq); err != nil {
			h.logger.Error("c.Bind")
			return httpErrors.ErrorCtxResponse(c, err)
		}
		hotelReq.HotelID = hotelUUID

		if err := h.validate.StructCtx(ctx, &hotelReq); err != nil {
			h.logger.Error("validate.StructCtx")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		hotel, err := h.service.UpdateHotel(ctx, &hotelReq)
		if err != nil {
			h.logger.Error("service.UpdateHotel")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, hotel)
	}
}

// Register GetHotelByID
// @Tags Hotels
// @Summary Get hotel by id
// @Description Get single hotel by uuid
// @Accept json
// @Produce json
// @Param hotel_id query string false "hotel uuid"
// @Success 200 {object} Hotel
// @Router /hotels/{hotel_id} [get]
// @BasePath /api/v1
func (h *serverImpl) GetHotelByID() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "serverImpl.GetHotelByID")
		defer span.Finish()

		hotelUUID, err := uuid.FromString(c.QueryParam("hotel_id"))
		if err != nil {
			h.logger.Error("uuid.FromString")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		hotelByID, err := h.service.GetHotelByID(ctx, hotelUUID)
		if err != nil {
			h.logger.Error("service.GetHotelByID")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, hotelByID)
	}
}

// Register GetHotels
// @Tags Hotels
// @Summary Get hotels list new user
// @Description Get hotels list with pagination using page and size query parameters
// @Accept json
// @Produce json
// @Param page query int false "page number"
// @Param size query int false "number of elements"
// @Success 200 {object} ListResult
// @Router /hotels [get]
// @BasePath /api/v1
func (h *serverImpl) GetHotels() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "serverImpl.GetHotels")
		defer span.Finish()

		page, err := strconv.Atoi(c.QueryParam("page"))
		if err != nil {
			h.logger.Error("strconv.Atoi")
			return httpErrors.ErrorCtxResponse(c, err)
		}
		size, err := strconv.Atoi(c.QueryParam("size"))
		if err != nil {
			h.logger.Error("strconv.Atoi")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		hotelsList, err := h.service.GetHotels(ctx, int64(page), int64(size))
		if err != nil {
			h.logger.Error("service.GetHotels")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.JSON(http.StatusOK, hotelsList)
	}
}

// UploadImage godoc
// @Summary Upload hotel image
// @Tags Hotels
// @Description Upload hotel logo image
// @Accept mpfd
// @Produce json
// @Param hotel_id query string false "hotel uuid"
// @Success 200 {object} Hotel
// @Router /hotels/{id}/image [put]
// @BasePath /api/v1
func (h *serverImpl) UploadImage() echo.HandlerFunc {
	bufferPool := &sync.Pool{New: func() interface{} {
		return &bytes.Buffer{}
	}}
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(c.Request().Context(), "serverImpl.UploadImage")
		defer span.Finish()

		hotelUUID, err := uuid.FromString(c.QueryParam("hotel_id"))
		if err != nil {
			return err
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

		fileType, err := CheckImageUpload(formFile)
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

		if err := h.service.UploadImage(ctx, buf.Bytes(), fileType, hotelUUID.String()); err != nil {
			h.logger.Error("service.UploadImage")
			return httpErrors.ErrorCtxResponse(c, err)
		}

		return c.NoContent(http.StatusOK)
	}
}

func (h *serverImpl) MapRoutes() {
	h.group.GET("", h.GetHotels())
	h.group.GET("/:hotel_id", h.GetHotelByID())
	h.group.POST("", h.CreateHotel(), h.mw.SessionMiddleware)
	h.group.PUT("/:hotel_id", h.UpdateHotel(), h.mw.SessionMiddleware)
	h.group.PUT("/:hotel_id/image", h.UploadImage(), h.mw.SessionMiddleware)
}

const (
	maxFileSize = 1024 * 1024 * 10
)

func CheckImageUpload(file multipart.File) (string, error) {
	fileHeader := make([]byte, maxFileSize)
	contentType := ""
	if _, err := file.Read(fileHeader); err != nil {
		return contentType, err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return contentType, err
	}

	count, err := file.Seek(0, 2)
	if err != nil {
		return contentType, err
	}
	if count > maxFileSize {
		return contentType, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return contentType, err
	}
	contentType = http.DetectContentType(fileHeader)

	if contentType != "image/jpg" && contentType != "image/png" && contentType != "image/jpeg" {
		return contentType, err
	}

	return contentType, nil
}
