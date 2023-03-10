package users

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"

	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/app/images"
	"github.com/badu/microservices-demo/app/sessions"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

const (
	imagesExchange = "images"
	resizeKey      = "resize_image_key"
	userUUIDHeader = "user_uuid"
)

type Repository interface {
	Create(ctx context.Context, user *UserDO) (*UserResponse, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	GetByEmail(ctx context.Context, email string) (*UserDO, error)
	Update(ctx context.Context, user *UserUpdate) (*UserResponse, error)
	UpdateAvatar(ctx context.Context, msg images.UploadedImageMsg) (*UserResponse, error)
	GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error)
}

type RedisRepository interface {
	SaveUser(ctx context.Context, user *UserResponse) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

type Publisher interface {
	CreateExchangeAndQueue(exchange, queueName, bindingKey string) (*amqp.Channel, error)
	Publish(ctx context.Context, exchange, routingKey, contentType string, headers amqp.Table, body []byte) error
}

type ServiceImpl struct {
	repository        Repository
	redisRepo         RedisRepository
	log               logger.Logger
	amqpPublisher     Publisher
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, sessions.AuthorizationServiceClient, error)
}

func NewService(
	repository Repository,
	redisRepository RedisRepository,
	log logger.Logger,
	amqpPublisher Publisher,
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, sessions.AuthorizationServiceClient, error),
) ServiceImpl {
	return ServiceImpl{
		repository:        repository,
		redisRepo:         redisRepository,
		log:               log,
		amqpPublisher:     amqpPublisher,
		grpcClientFactory: grpcClientFactory,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.GetByID")
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
		return nil, errors.Join(err, errors.New("getting user by id in service"))
	}

	if err := s.redisRepo.SaveUser(ctx, userResponse); err != nil {
		s.log.Errorf("redisRepo.SaveUser: %v", err)
	}

	return userResponse, nil
}

func (s *ServiceImpl) Register(ctx context.Context, user *UserDO) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.Register")
	defer span.Finish()

	if err := user.PrepareCreate(); err != nil {
		return nil, errors.Join(err, errors.New("preparing user creation in service"))
	}

	created, err := s.repository.Create(ctx, user)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating user in service"))
	}

	return created, err
}

func (s *ServiceImpl) Login(ctx context.Context, login Login) (*UserDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.Login")
	defer span.Finish()

	userByEmail, err := s.repository.GetByEmail(ctx, login.Email)
	if err != nil {
		return nil, errors.Join(err, errors.New("getting user by email in service"))
	}

	if err := userByEmail.ComparePasswords(login.Password); err != nil {
		return nil, errors.Join(err, errors.New("passwords not equal in service"))
	}

	userByEmail.SanitizePassword()

	return userByEmail, nil
}

func (s *ServiceImpl) CreateSession(ctx context.Context, userID uuid.UUID) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.CreateSession")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return "", errors.Join(err, errors.New("creating grpc client for user session in service"))
	}
	defer conn.Close()

	session, err := client.CreateSession(ctx, &sessions.CreateSessionRequest{UserID: userID.String()})
	if err != nil {
		return "", errors.Join(err, errors.New("calling create session over grpc client"))
	}

	return session.GetSession().GetSessionID(), err
}

func (s *ServiceImpl) DeleteSession(ctx context.Context, sessionID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.DeleteSession")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return errors.Join(err, errors.New("creating grpc client for deleting session in service"))
	}
	defer conn.Close()

	_, err = client.DeleteSession(ctx, &sessions.DeleteSessionRequest{SessionID: sessionID})
	if err != nil {
		return errors.Join(err, errors.New("calling grpc client for deleting session"))
	}

	return nil
}

func (s *ServiceImpl) GetSessionByID(ctx context.Context, sessionID string) (*sessions.SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.GetSessionByID")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating grpc client for retrieving session"))
	}
	defer conn.Close()

	sessionByID, err := client.GetSessionByID(ctx, &sessions.GetSessionByIDRequest{SessionID: sessionID})
	if err != nil {
		return nil, errors.Join(err, errors.New("while getting session by id from grpc client"))
	}

	sess := &sessions.SessionDO{}
	sess, err = sess.FromProto(sessionByID.GetSession())
	if err != nil {
		return nil, errors.Join(err, errors.New("while transforming grpc client response"))
	}

	return sess, nil
}

func (s *ServiceImpl) GetCSRFToken(ctx context.Context, sessionID string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.GetCSRFToken")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return "", errors.Join(err, errors.New("creating grpc client"))
	}
	defer conn.Close()

	csrfToken, err := client.CreateCsrfToken(
		ctx,
		&sessions.CreateCsrfTokenRequest{CsrfTokenInput: &sessions.CsrfTokenInput{SessionID: sessionID}},
	)
	if err != nil {
		return "", errors.Join(err, errors.New("sessionsClient.CreateCsrfToken"))
	}

	return csrfToken.GetCsrfToken().GetToken(), nil
}

func (s *ServiceImpl) Update(ctx context.Context, user *UserUpdate) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.Update")
	defer span.Finish()

	ctxUser, ok := ctx.Value(RequestCtxUser{}).(*UserResponse)
	if !ok {
		return nil, errors.Join(httpErrors.Unauthorized, errors.New("user not present in context"))
	}

	if ctxUser.UserID != user.UserID || *ctxUser.Role != RoleAdmin {
		return nil, errors.Join(httpErrors.WrongCredentials, errors.New("user is not owner or admin"))
	}

	userResponse, err := s.repository.Update(ctx, user)
	if err != nil {
		return nil, errors.Join(err, errors.New("service.Update.userPGRepo.Update"))
	}

	if err := s.redisRepo.SaveUser(ctx, userResponse); err != nil {
		s.log.Errorf("redisRepo.SaveUser: %v", err)
	}

	return userResponse, nil
}

func (s *ServiceImpl) UpdateUploadedAvatar(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.UpdateUploadedAvatar")
	defer span.Finish()

	var img images.ImageDO
	if err := json.Unmarshal(delivery.Body, &img); err != nil {
		return errors.Join(err, errors.New("UpdateUploadedAvatar.json.Unmarshal"))
	}

	userUUID, ok := delivery.Headers[userUUIDHeader].(string)
	if !ok {
		return errors.Join(httpErrors.InvalidUUID, errors.New("delivery.Headers"))
	}

	uid, err := uuid.FromString(userUUID)
	if err != nil {
		return errors.Join(err, errors.New("reading uuid"))
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

func (s *ServiceImpl) UpdateAvatar(ctx context.Context, data *images.UpdateAvatarMsg) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.UpdateAvatar")
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
		return errors.Join(err, errors.New("publishing avatar updated"))
	}

	s.log.Infof("Publish UpdateAvatar %-v", headers)
	return nil
}

func (s *ServiceImpl) GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_service.GetUsersByIDs")
	defer span.Finish()
	return s.repository.GetUsersByIDs(ctx, userIDs)
}
