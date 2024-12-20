package goarpa_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/erfandiakoo/goarpa/v2"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

type configAdmin struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type configUser struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	HostName string      `json:"hostname"`
	Proxy    string      `json:"proxy,omitempty"`
	Admin    configAdmin `json:"admin"`
	User     configUser  `json:"user"`
}

var (
	config     *Config
	configOnce sync.Once
	setupOnce  sync.Once
	testUserID string
)

type RestyLogWriter struct {
	io.Writer
	t testing.TB
}

func (w *RestyLogWriter) Errorf(format string, v ...interface{}) {
	w.write("[ERROR] "+format, v...)
}

func (w *RestyLogWriter) Warnf(format string, v ...interface{}) {
	w.write("[WARN] "+format, v...)
}

func (w *RestyLogWriter) Debugf(format string, v ...interface{}) {
	w.write("[DEBUG] "+format, v...)
}

func (w *RestyLogWriter) write(format string, v ...interface{}) {
	w.t.Logf(format, v...)
}

func GetConfig(t testing.TB) *Config {
	configOnce.Do(func() {
		rand.Seed(uint64(time.Now().UTC().UnixNano()))
		configFileName, ok := os.LookupEnv("GOARPA_TEST_CONFIG")
		if !ok {
			configFileName = filepath.Join("testdata", "config.json")
		}
		configFile, err := os.Open(configFileName)
		require.NoError(t, err, "cannot open config.json")
		defer func() {
			err := configFile.Close()
			require.NoError(t, err, "cannot close config file")
		}()
		data, err := ioutil.ReadAll(configFile)
		require.NoError(t, err, "cannot read config.json")
		config = &Config{}
		err = json.Unmarshal(data, config)
		require.NoError(t, err, "cannot parse config.json")
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		if len(config.Proxy) != 0 {
			proxy, err := url.Parse(config.Proxy)
			require.NoError(t, err, "incorrect proxy url: "+config.Proxy)
			http.DefaultTransport.(*http.Transport).Proxy = http.ProxyURL(proxy)
		}
	})
	return config
}

func NewClientWithDebug(t testing.TB) *goarpa.GoArpa {
	cfg := GetConfig(t)
	client := goarpa.NewClient(cfg.HostName)
	cond := func(resp *resty.Response, err error) bool {
		if resp != nil && resp.IsError() {
			if e, ok := resp.Error().(*goarpa.HTTPErrorResponse); ok {
				msg := e.String()
				return strings.Contains(msg, "Cached clientScope not found") || strings.Contains(msg, "unknown_error")
			}
		}
		return false
	}

	restyClient := client.RestyClient()

	// restyClient.AddRetryCondition(
	// 	func(r *resty.Response, err error) bool {
	// 		if err != nil || r.RawResponse.StatusCode == 500 || r.RawResponse.StatusCode == 502 {
	// 			return true
	// 		}

	// 		return false
	// 	},
	// ).SetRetryCount(5).SetRetryWaitTime(10 * time.Millisecond)

	restyClient.
		// SetDebug(true).
		SetLogger(&RestyLogWriter{
			t: t,
		}).
		SetRetryCount(10).
		SetRetryWaitTime(2 * time.Second).
		AddRetryCondition(cond)

	return client
}

// FailRequest fails requests and returns an error
//
//	err - returned error or nil to return the default error
//	failN - number of requests to be failed
//	skipN = number of requests to be executed and not failed by this function
func FailRequest(client *goarpa.GoArpa, err error, failN, skipN int) *goarpa.GoArpa {
	client.RestyClient().OnBeforeRequest(
		func(c *resty.Client, r *resty.Request) error {
			if skipN > 0 {
				skipN--
				return nil
			}
			if failN == 0 {
				return nil
			}
			failN--
			if err == nil {
				err = fmt.Errorf("an error for request: %+v", r)
			}
			return err
		},
	)
	return client
}

func GetToken(t testing.TB, client *goarpa.GoArpa) (string, []*http.Cookie) {
	cfg := GetConfig(t)
	token, cookie, err := client.GetAdminToken(
		context.Background(),
		cfg.Admin.UserName,
		cfg.Admin.Password,
	)
	require.NoError(t, err, "Login failed")
	require.NotEmpty(t, token, "Got an empty token")
	return token, cookie
}

// ---------
// API tests
// ---------

func Test_GetAdminToken(t *testing.T) {
	t.Parallel()
	cfg := GetConfig(t)
	client := NewClientWithDebug(t)

	// Obtain the token from AdminAuthenticate
	newToken, cookie, err := client.GetAdminToken(
		context.Background(),
		cfg.Admin.UserName,
		cfg.Admin.Password,
	)

	require.NoError(t, err, "Login failed")
	require.NotEmpty(t, newToken, "Got an empty token")
	require.NotEmpty(t, cookie, "Got an empty cookie")

	t.Logf("New token: %s", newToken)
	t.Logf("New cookie: %s", cookie)
}

func Test_GetCustomerByMobile(t *testing.T) {
	t.Parallel()
	client := NewClientWithDebug(t)
	token, cookie := GetToken(t, client)

	customerInfo, err := client.GetCustomerByMobile(
		context.Background(),
		token,
		cookie,
		"09128575183",
	)
	require.NoError(t, err, "Expected no error when fetching valid customer info")
	require.NotNil(t, customerInfo, "Expected customer info, got nil")
	t.Logf("Customer Info: %+v", customerInfo)

	FailRequest(client, nil, 1, 0)

	_, err = client.GetCustomerByMobile(
		context.Background(),
		token,
		cookie,
		"09128575183",
	)
	require.Error(t, err, "Expected an error when request fails")

	assert.Contains(t, err.Error(), "could not get customer info", "Error message mismatch")
}

func Test_GetCustomerByBusinessCode(t *testing.T) {
	t.Parallel()
	client := NewClientWithDebug(t)
	token, cookie := GetToken(t, client)

	customerInfo, err := client.GetCustomerByBusinessCode(
		context.Background(),
		token,
		cookie,
		"127013",
	)
	require.NoError(t, err, "Expected no error when fetching valid customer info")
	require.NotNil(t, customerInfo, "Expected customer info, got nil")
	t.Logf("Customer Info: %+v", customerInfo)

	FailRequest(client, nil, 1, 0)

	_, err = client.GetCustomerByBusinessCode(
		context.Background(),
		token,
		cookie,
		"127013",
	)
	require.Error(t, err, "Expected an error when request fails")

	assert.Contains(t, err.Error(), "could not get customer info", "Error message mismatch")
}

func Test_GetServiceByItemCode(t *testing.T) {
	t.Parallel()
	client := NewClientWithDebug(t)
	token, cookie := GetToken(t, client)

	customerInfo, err := client.GetServiceByItemCode(
		context.Background(),
		token,
		cookie,
		"650304",
	)
	require.NoError(t, err, "Expected no error when fetching valid service info")
	require.NotNil(t, customerInfo, "Expected service info, got nil")
	t.Logf("Service Info: %+v", customerInfo)

	FailRequest(client, nil, 1, 0)

	_, err = client.GetServiceByItemCode(
		context.Background(),
		token,
		cookie,
		"650304",
	)
	require.Error(t, err, "Expected an error when request fails")

	assert.Contains(t, err.Error(), "could not get service info", "Error message mismatch")
}
