package rtvbp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/babelforce/rtvbp-go/proto"
	"io"
	"log/slog"
)

type SessionHandler interface {
	OnBegin(ctx context.Context, h SHC) error
	OnRequest(ctx context.Context, h SHC, req *proto.Request) error
	OnEvent(ctx context.Context, h SHC, evt *proto.Event) error
}

type HandlerAudio interface {
	io.ReadWriter
	ClearBuffer() (int, error)
	Len() int
}

// SHC - Session Handler Context
type SHC interface {
	// SessionID retrieves the ID of the current session
	SessionID() string

	// Log returns the sessions Logger
	Log() *slog.Logger

	// Request performs a request execution
	Request(ctx context.Context, req NamedRequest) (*proto.Response, error)
	Respond(ctx context.Context, res *proto.Response) error
	Notify(ctx context.Context, evt NamedEvent) error
	AudioStream() HandlerAudio

	Close(ctx context.Context) error

	State() SessionState
}

type RequestHooks interface {
	// OnAfterReply is called when a handler has replied successfully to a request
	OnAfterReply(ctx context.Context, hc SHC) error
}

type RequestHooksAdapter struct {
}

func (r *RequestHooksAdapter) OnAfterReply(_ context.Context, hc SHC) error {
	return nil
}

var _ RequestHooks = &RequestHooksAdapter{}

type sessionHandlerCtx struct {
	sess *Session
	ha   HandlerAudio
}

func (shc *sessionHandlerCtx) State() SessionState {
	return shc.sess.State()
}

func (shc *sessionHandlerCtx) AudioStream() HandlerAudio {
	return shc.ha
}

func (shc *sessionHandlerCtx) Respond(ctx context.Context, res *proto.Response) error {
	data, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	if err := shc.sess.writeMsgData(data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (shc *sessionHandlerCtx) Close(ctx context.Context) error {
	return shc.sess.Close(ctx)
}

func (shc *sessionHandlerCtx) SessionID() string {
	return shc.sess.id
}

func (shc *sessionHandlerCtx) Log() *slog.Logger {
	return shc.sess.logger
}

func (shc *sessionHandlerCtx) Request(ctx context.Context, req NamedRequest) (*proto.Response, error) {
	return shc.sess.Request(ctx, req)
}

func (shc *sessionHandlerCtx) Notify(ctx context.Context, evt NamedEvent) error {
	return shc.sess.Notify(ctx, evt)
}

var _ SHC = &sessionHandlerCtx{}

type defaultSessionHandler struct {
	eventHandlers   map[string]EventHandler
	requestHandlers map[string]RequestHandler
	onEnd           func(ctx context.Context, h SHC) error
	onBegin         func(ctx context.Context, h SHC) error
}

func (d *defaultSessionHandler) OnBegin(ctx context.Context, hc SHC) error {
	if d.onBegin != nil {
		return d.onBegin(ctx, hc)
	}
	return nil
}

func (d *defaultSessionHandler) OnEnd(ctx context.Context, hc SHC) error {
	if d.onEnd != nil {
		return d.onEnd(ctx, hc)
	}
	return nil
}

func (d *defaultSessionHandler) OnRequest(ctx context.Context, hc SHC, req *proto.Request) error {
	hdl, ok := d.requestHandlers[req.Method]
	if !ok {
		return hc.Respond(ctx, req.NotOk(proto.NewError(501, fmt.Sprintf("unknown method: %s", req.Method))))
	}

	return hdl.Handle(ctx, hc, req)
}

func (d *defaultSessionHandler) OnEvent(ctx context.Context, hc SHC, evt *proto.Event) error {
	hdl, ok := d.eventHandlers[evt.Event]
	if !ok {
		return nil
	}

	return hdl.Handle(ctx, hc, evt)
}

type HandlerConfig struct {
	OnBegin func(ctx context.Context, h SHC) error
}

// NewHandler creates a new handler
func NewHandler(config HandlerConfig, args ...any) SessionHandler {
	handler := &defaultSessionHandler{
		eventHandlers:   make(map[string]EventHandler),
		requestHandlers: make(map[string]RequestHandler),
		onBegin:         config.OnBegin,
	}

	// add handlers from args
	for _, arg := range args {
		switch arg := arg.(type) {
		case EventHandler:
			handler.eventHandlers[arg.EventName()] = arg
		case RequestHandler:
			handler.requestHandlers[arg.MethodName()] = arg
		}
	}

	return handler
}

var _ SessionHandler = &defaultSessionHandler{}
