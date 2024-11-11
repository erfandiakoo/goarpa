package goarpa

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

type GoArpa struct {
	basePath    string
	restyClient *resty.Client
	Config      struct {
		GetServiceTokenEndpoint string
	}
}

const (
	adminClientID string = "admin-cli"
	urlSeparator  string = "/"
)

func makeURL(path ...string) string {
	return strings.Join(path, urlSeparator)
}

// GetRequest returns a request for calling endpoints.
func (g *GoArpa) GetRequest(ctx context.Context) *resty.Request {
	var err HTTPErrorResponse
	return injectTracingHeaders(
		ctx, g.restyClient.R().
			SetContext(ctx).
			SetError(&err),
	)
}

func injectTracingHeaders(ctx context.Context, req *resty.Request) *resty.Request {
	// look for span in context, do nothing if span is not found
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return req
	}

	// look for tracer in context, use global tracer if not found
	tracer, ok := ctx.Value(tracerContextKey).(opentracing.Tracer)
	if !ok || tracer == nil {
		tracer = opentracing.GlobalTracer()
	}

	// inject tracing header into request
	err := tracer.Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))
	if err != nil {
		return req
	}

	return req
}

// GetRequestWithBearerAuthNoCache returns a JSON base request configured with an auth token and no-cache header.
func (g *GoArpa) GetRequestWithBearerAuthNoCache(ctx context.Context, token string) *resty.Request {
	return g.GetRequest(ctx).
		SetAuthToken(token).
		SetHeader("Content-Type", "application/json").
		SetHeader("Cache-Control", "no-cache")
}

// GetRequestWithBearerAuth returns a JSON base request configured with an auth token.
func (g *GoArpa) GetRequestWithBearerAuth(ctx context.Context, token string) *resty.Request {
	return g.GetRequest(ctx).
		SetAuthToken(token).
		SetHeader("Content-Type", "application/json")
}

func NewClient(basePath string, options ...func(*GoArpa)) *GoArpa {
	c := GoArpa{
		basePath:    strings.TrimRight(basePath, urlSeparator),
		restyClient: resty.New(),
	}

	c.Config.GetServiceTokenEndpoint = makeURL("serv", "token", "GetServiceToken")

	for _, option := range options {
		option(&c)
	}

	return &c
}

// RestyClient returns the internal resty g.
// This can be used to configure the g.
func (g *GoArpa) RestyClient() *resty.Client {
	return g.restyClient
}

// SetRestyClient overwrites the internal resty g.
func (g *GoArpa) SetRestyClient(restyClient *resty.Client) {
	g.restyClient = restyClient
}

func checkForError(resp *resty.Response, err error, errMessage string) error {
	if err != nil {
		return &APIError{
			Code:    0,
			Message: errors.Wrap(err, errMessage).Error(),
			Type:    ParseAPIErrType(err),
		}
	}

	if resp == nil {
		return &APIError{
			Message: "empty response",
			Type:    ParseAPIErrType(err),
		}
	}

	if resp.IsError() {
		var msg string

		if e, ok := resp.Error().(*HTTPErrorResponse); ok && e.NotEmpty() {
			msg = fmt.Sprintf("%s: %s", resp.Status(), e)
		} else {
			msg = resp.Status()
		}

		return &APIError{
			Code:    resp.StatusCode(),
			Message: msg,
			Type:    ParseAPIErrType(err),
		}
	}

	return nil
}

func (g *GoArpa) AdminAuthenticate(ctx context.Context, username string, password string) (*JWT, error) {
	const errMessage = "could not get token"

	var token JWT
	var req *resty.Request

	req = g.GetRequest(ctx)

	resp, err := req.SetHeaders(map[string]string{
		"username": username,
		"password": password,
	}).
		SetResult(&token).
		Post(g.basePath + "/" + g.Config.GetServiceTokenEndpoint)

	if err := checkForError(resp, err, errMessage); err != nil {
		return nil, err
	}

	return &token, nil
}
