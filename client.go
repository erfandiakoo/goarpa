package goarpa

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/erfandiakoo/goarpa/shared/constant"
	"github.com/go-resty/resty/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

type GoArpa struct {
	basePath    string
	restyClient *resty.Client
	Config      struct {
		GetServiceTokenEndpoint   string
		CreateCustomerEndpoint    string
		CreateTransactionEndpoint string
		CreateServiceEndpoint     string
		GetCustomerEndpoint       string
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

func (g *GoArpa) GetRequestWithBearerAuthWithCookie(ctx context.Context, token string, cookie []*http.Cookie) *resty.Request {
	return g.GetRequest(ctx).
		SetAuthToken(token).
		SetCookie(cookie[0]).
		SetHeader("Content-Type", "application/json")
}

func NewClient(basePath string, options ...func(*GoArpa)) *GoArpa {
	c := GoArpa{
		basePath:    strings.TrimRight(basePath, urlSeparator),
		restyClient: resty.New(),
	}

	c.Config.GetServiceTokenEndpoint = makeURL("serv", "token", "GetServiceToken")
	c.Config.CreateCustomerEndpoint = makeURL("serv", "api", "PostBussiness")
	c.Config.CreateTransactionEndpoint = makeURL("serv", "api", "NewTransaction")
	c.Config.CreateServiceEndpoint = makeURL("serv", "api", "PostService")
	c.Config.GetCustomerEndpoint = makeURL("serv", "api", "GetBusiness")

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

func (g *GoArpa) GetAdminToken(ctx context.Context, username string, password string) (string, []*http.Cookie, error) {
	const errMessage = "could not get token"

	req := g.GetRequest(ctx)

	resp, err := req.SetQueryParams(map[string]string{
		"username": username,
		"password": password,
	}).
		Get(g.basePath + "/" + g.Config.GetServiceTokenEndpoint + "?")

	if err := checkForError(resp, err, errMessage); err != nil {
		return "", nil, err
	}

	return resp.String(), resp.Cookies(), nil
}

func (g *GoArpa) CreateCustomer(ctx context.Context, accessToken string, customer CreateCustomerRequest) (*CreateCustomerResponse, error) {
	const errMessage = "could not create customer"

	var response CreateCustomerResponse

	resp, err := g.GetRequestWithBearerAuth(ctx, accessToken).
		SetBody(customer).
		SetResult(response).
		Post(g.basePath + "/" + g.Config.CreateCustomerEndpoint)

	if err := checkForError(resp, err, errMessage); err != nil {
		return nil, err
	}

	return &response, nil
}

func (g *GoArpa) CreateTransaction(ctx context.Context, accessToken string, transaction CreateTransactionRequest) (*CreateTransactionResponse, error) {
	const errMessage = "could not create transaction"

	var response CreateTransactionResponse

	resp, err := g.GetRequestWithBearerAuth(ctx, accessToken).
		SetBody(transaction).
		SetResult(response).
		Post(g.basePath + "/" + g.Config.CreateTransactionEndpoint)

	if err := checkForError(resp, err, errMessage); err != nil {
		return nil, err
	}

	return &response, nil
}

func (g *GoArpa) CreateService(ctx context.Context, accessToken string, service CreateServiceRequest) (*CreateServiceResponse, error) {
	const errMessage = "could not create service"

	var response CreateServiceResponse

	resp, err := g.GetRequestWithBearerAuth(ctx, accessToken).
		SetBody(service).
		SetResult(response).
		Post(g.basePath + "/" + g.Config.CreateServiceEndpoint)

	if err := checkForError(resp, err, errMessage); err != nil {
		return nil, err
	}

	return &response, nil
}

func (g *GoArpa) GetCustomerByMobile(ctx context.Context, accessToken string, cookie []*http.Cookie, mobile string) (*GetCustomerResponse, error) {
	const errMessage = "could not get customer info"

	// Create an instance of GetCustomerResponse to hold the response
	result := &GetCustomerResponse{}

	// Make the request and set result to auto-unmarshal
	resp, err := g.GetRequestWithBearerAuthWithCookie(ctx, accessToken, cookie).
		SetQueryParam(constant.MobileKey, mobile).
		SetResult(result).
		Get(fmt.Sprintf("%s/%s", g.basePath, g.Config.GetCustomerEndpoint))

	// Check for errors
	if err := checkForError(resp, err, errMessage); err != nil {
		return nil, err
	}

	// Return the unmarshaled result
	return result, nil
}

func (g *GoArpa) GetCustomerByBusinessCode(ctx context.Context, accessToken string, cookie []*http.Cookie, businessCode string) (*GetCustomerResponse, error) {
	const errMessage = "could not get customer info"

	result := &GetCustomerResponse{}

	resp, err := g.GetRequestWithBearerAuthWithCookie(ctx, accessToken, cookie).
		SetQueryParam(constant.BusinessCodeKey, businessCode).
		SetResult(result).
		Get(fmt.Sprintf("%s/%s", g.basePath, g.Config.GetCustomerEndpoint))

	if err := checkForError(resp, err, errMessage); err != nil {
		return nil, err
	}

	return result, nil
}
