package events

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	"github.com/badu/microservices-demo/app/comments"
	"github.com/badu/microservices-demo/app/hotels"
	"github.com/badu/microservices-demo/app/sessions"
	"github.com/badu/microservices-demo/app/users"
)

type commonGRPCRequestEvent struct {
	Ctx  context.Context
	wg   *sync.WaitGroup
	Err  error
	Conn *grpc.ClientConn
}

func (r *commonGRPCRequestEvent) Async() bool {
	return false
}

func (r *commonGRPCRequestEvent) Reply() {
	r.wg.Done()
}

func (r *commonGRPCRequestEvent) WaitReply() {
	r.wg.Wait()
}

func NewRequireHotelsGRPCClient(ctx context.Context) *RequireHotelsGRPCClient {
	var wg sync.WaitGroup
	wg.Add(1)
	result := RequireHotelsGRPCClient{commonGRPCRequestEvent: commonGRPCRequestEvent{Ctx: ctx, wg: &wg}}
	return &result
}

type RequireHotelsGRPCClient struct {
	commonGRPCRequestEvent
	Client hotels.HotelsServiceClient
}

func (r *RequireHotelsGRPCClient) EventID() string {
	return "RequireHotelsGRPCClientEvent"
}

func NewRequireCommentsGRPCClient(ctx context.Context) *RequireCommentsGRPCClient {
	var wg sync.WaitGroup
	wg.Add(1)
	result := RequireCommentsGRPCClient{commonGRPCRequestEvent: commonGRPCRequestEvent{Ctx: ctx, wg: &wg}}
	return &result
}

type RequireCommentsGRPCClient struct {
	commonGRPCRequestEvent
	Client comments.CommentsServiceClient
}

func (r *RequireCommentsGRPCClient) EventID() string {
	return "RequireCommentsGRPCClientEvent"
}

func NewRequireUsersGRPCClient(ctx context.Context) *RequireUsersGRPCClient {
	var wg sync.WaitGroup
	wg.Add(1)
	result := RequireUsersGRPCClient{commonGRPCRequestEvent: commonGRPCRequestEvent{Ctx: ctx, wg: &wg}}
	return &result
}

type RequireUsersGRPCClient struct {
	commonGRPCRequestEvent
	Client users.UserServiceClient
}

func (r *RequireUsersGRPCClient) EventID() string {
	return "RequireUsersGRPCClientEvent"
}

func NewRequireSessionsGRPCClient(ctx context.Context) *RequireSessionsGRPCClient {
	var wg sync.WaitGroup
	wg.Add(1)
	result := RequireSessionsGRPCClient{commonGRPCRequestEvent: commonGRPCRequestEvent{Ctx: ctx, wg: &wg}}
	return &result
}

type RequireSessionsGRPCClient struct {
	commonGRPCRequestEvent
	Client sessions.AuthorizationServiceClient
}

func (r *RequireSessionsGRPCClient) EventID() string {
	return "RequireSessionsGRPCClientEvent"
}
