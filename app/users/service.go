package users

import (
	"context"
	"encoding/json"

	"github.com/badu/microservices-demo/app/images"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/app/sessions"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	Register(ctx context.Context, user *UserDO) (*UserResponse, error)
	Login(ctx context.Context, login Login) (*UserDO, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	CreateSession(ctx context.Context, userID uuid.UUID) (string, error)
	GetSessionByID(ctx context.Context, sessionID string) (*sessions.SessionDO, error)
	GetCSRFToken(ctx context.Context, sessionID string) (string, error)
	DeleteSession(ctx context.Context, sessionID string) error
	Update(ctx context.Context, user *UserUpdate) (*UserResponse, error)
	UpdateUploadedAvatar(ctx context.Context, delivery amqp.Delivery) error
	UpdateAvatar(ctx context.Context, data *images.UpdateAvatarMsg) error
	GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error)
}

const (
	imagesExchange = "images"
	resizeKey      = "resize_image_key"
	userUUIDHeader = "user_uuid"
)

type serviceImpl struct {
	repository      Repository
	authServicePort string
	redisRepo       RedisRepository
	log             logger.Logger
	amqpPublisher   Publisher
	mw              *grpc_client.ClientMiddleware
}

func NewService(
	repository Repository,
	redisRepository RedisRepository,
	log logger.Logger,
	amqpPublisher Publisher,
	cfg *config.Config,
	tracer opentracing.Tracer,
) *serviceImpl {
	return &serviceImpl{
		repository:      repository,
		redisRepo:       redisRepository,
		log:             log,
		amqpPublisher:   amqpPublisher,
		authServicePort: cfg.GRPC.SessionServicePort,
		mw:              grpc_client.NewClientMiddleware(log, cfg, tracer),
	}
}

func (s *serviceImpl) GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.GetByID")
	defer span.Finish()

	cachedUser, err := s.redisRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.log.Errorf("redisRepo.GetUserByID: %v", err)
	}
	if cachedUser != nil {
		return cachedUser, nil
	}

	userResponse, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "service.userPGRepo.GetByID")
	}

	if err := s.redisRepo.SaveUser(ctx, userResponse); err != nil {
		s.log.Errorf("redisRepo.SaveUser: %v", err)
	}

	return userResponse, nil
}

func (s *serviceImpl) Register(ctx context.Context, user *UserDO) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.Register")
	defer span.Finish()

	if err := user.PrepareCreate(); err != nil {
		return nil, errors.Wrap(err, "user.PrepareCreate")
	}

	created, err := s.repository.Create(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "userPGRepo.Create")
	}

	return created, err
}

func (s *serviceImpl) Login(ctx context.Context, login Login) (*UserDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.Login")
	defer span.Finish()

	userByEmail, err := s.repository.GetByEmail(ctx, login.Email)
	if err != nil {
		return nil, errors.Wrap(err, "userPGRepo.GetByEmail")
	}

	if err := userByEmail.ComparePasswords(login.Password); err != nil {
		return nil, errors.Wrap(err, "service.ComparePasswords")
	}

	userByEmail.SanitizePassword()

	return userByEmail, nil
}

func (s *serviceImpl) CreateSession(ctx context.Context, userID uuid.UUID) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.CreateSession")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.authServicePort)
	if err != nil {
		return "", errors.Wrap(err, "serviceImpl.CreateSession")
	}
	defer conn.Close()

	client := sessions.NewAuthorizationServiceClient(conn)

	session, err := client.CreateSession(ctx, &sessions.CreateSessionRequest{UserID: userID.String()})
	if err != nil {
		return "", errors.Wrap(err, "sessionsClient.CreateSession")
	}

	return session.GetSession().GetSessionID(), err
}

func (s *serviceImpl) DeleteSession(ctx context.Context, sessionID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.DeleteSession")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.authServicePort)
	if err != nil {
		return errors.Wrap(err, "serviceImpl.DeleteSession")
	}
	defer conn.Close()

	client := sessions.NewAuthorizationServiceClient(conn)

	_, err = client.DeleteSession(ctx, &sessions.DeleteSessionRequest{SessionID: sessionID})
	if err != nil {
		return errors.Wrap(err, "sessionsClient.DeleteSession")
	}

	return nil
}

func (s *serviceImpl) GetSessionByID(ctx context.Context, sessionID string) (*sessions.SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.GetSessionByID")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.authServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "serviceImpl.DeleteSession")
	}
	defer conn.Close()

	client := sessions.NewAuthorizationServiceClient(conn)

	sessionByID, err := client.GetSessionByID(ctx, &sessions.GetSessionByIDRequest{SessionID: sessionID})
	if err != nil {
		return nil, errors.Wrap(err, "sessionsClient.GetSessionByID")
	}

	sess := &sessions.SessionDO{}
	sess, err = sess.FromProto(sessionByID.GetSession())
	if err != nil {
		return nil, errors.Wrap(err, "sess.FromProto")
	}

	return sess, nil
}

func (s *serviceImpl) GetCSRFToken(ctx context.Context, sessionID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.GetCSRFToken")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.authServicePort)
	if err != nil {
		return "", errors.Wrap(err, "serviceImpl.DeleteSession")
	}
	defer conn.Close()

	client := sessions.NewAuthorizationServiceClient(conn)

	csrfToken, err := client.CreateCsrfToken(
		ctx,
		&sessions.CreateCsrfTokenRequest{CsrfTokenInput: &sessions.CsrfTokenInput{SessionID: sessionID}},
	)
	if err != nil {
		return "", errors.Wrap(err, "sessionsClient.CreateCsrfToken")
	}

	return csrfToken.GetCsrfToken().GetToken(), nil
}

func (s *serviceImpl) Update(ctx context.Context, user *UserUpdate) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.Update")
	defer span.Finish()

	ctxUser, ok := ctx.Value(RequestCtxUser{}).(*UserResponse)
	if !ok {
		return nil, errors.Wrap(httpErrors.Unauthorized, "ctx.Value user")
	}

	if ctxUser.UserID != user.UserID || *ctxUser.Role != RoleAdmin {
		return nil, errors.Wrap(httpErrors.WrongCredentials, "user is not owner or admin")
	}

	userResponse, err := s.repository.Update(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "service.Update.userPGRepo.Update")
	}

	if err := s.redisRepo.SaveUser(ctx, userResponse); err != nil {
		s.log.Errorf("redisRepo.SaveUser: %v", err)
	}

	return userResponse, nil
}

func (s *serviceImpl) UpdateUploadedAvatar(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.UpdateUploadedAvatar")
	defer span.Finish()

	var img images.ImageDO
	if err := json.Unmarshal(delivery.Body, &img); err != nil {
		return errors.Wrap(err, "UpdateUploadedAvatar.json.Unmarshal")
	}

	userUUID, ok := delivery.Headers[userUUIDHeader].(string)
	if !ok {
		return errors.Wrap(httpErrors.InvalidUUID, "delivery.Headers")
	}

	uid, err := uuid.FromString(userUUID)
	if err != nil {
		return errors.Wrap(err, "uuid.FromString")
	}

	created, err := s.repository.UpdateAvatar(ctx, images.UploadedImageMsg{
		ImageID:    img.ImageID,
		UserID:     uid,
		ImageURL:   img.ImageURL,
		IsUploaded: img.IsUploaded,
	})
	if err != nil {
		return err
	}

	s.log.Infof("UpdateUploadedAvatar: %s", created.Avatar)

	return nil
}

func (s *serviceImpl) UpdateAvatar(ctx context.Context, data *images.UpdateAvatarMsg) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.UpdateAvatar")
	defer span.Finish()

	headers := make(amqp.Table, 1)
	headers[userUUIDHeader] = data.UserID.String()
	if err := s.amqpPublisher.Publish(
		ctx,
		imagesExchange,
		resizeKey,
		data.ContentType,
		headers,
		data.Body,
	); err != nil {
		return errors.Wrap(err, "UpdateUploadedAvatar.Publish")
	}

	s.log.Infof("Publish UpdateAvatar %-v", headers)
	return nil
}

func (s *serviceImpl) GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.GetUsersByIDs")
	defer span.Finish()
	return s.repository.GetUsersByIDs(ctx, userIDs)
}
