// In this package, we are using Miniredis as we expect all of the commands to function properly without errors.
package application

import (
	"encoding/base32"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
)

// Mock Redis dependency.
func initializeTestRedis() *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatal(err.Error())
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client
}

func structToJSON(object interface{}) string {
	out, err := json.Marshal(object)
	if err != nil {
		log.Fatal(err.Error())
	}

	return string(out)
}

func TestGeneralHandlers(t *testing.T) {
	rdb := initializeTestRedis()
	handler := Configure(rdb)
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	failureTests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
		expectedBody   *FailureResponse
	}{
		{
			name:           "test_failure_not_found",
			method:         http.MethodGet,
			route:          "/api/v1/404",
			expectedStatus: http.StatusNotFound,
			expectedBody:   NewFailureResponse(http.StatusNotFound, "Route '/api/v1/404' with method 'GET' does not exist in this server!"),
		},
		{
			name:           "test_failure_method_not_allowed",
			method:         http.MethodDelete,
			route:          "/api/v1",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   NewFailureResponse(http.StatusMethodNotAllowed, "Method 'DELETE' is not allowed in this route!"),
		},
	}

	successTests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
		expectedBody   *SuccessResponse
	}{
		{
			name:           "test_health_check",
			method:         http.MethodGet,
			route:          "/api/v1",
			expectedStatus: http.StatusOK,
			expectedBody:   NewSuccessResponse(http.StatusOK, "Welcome to 'net/http' API!", nil),
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			assert.NotNil(t, w.Body)
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, structToJSON(tt.expectedBody), w.Body.String())
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			assert.NotNil(t, w.Body)
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, structToJSON(tt.expectedBody), w.Body.String())
		})
	}
}

func TestDecodeJSONBody(t *testing.T) {
	rdb := initializeTestRedis()
	handler := Configure(rdb)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	tests := []struct {
		name         string
		method       string
		route        string
		input        string
		expectedBody *FailureResponse
		withHeader   bool
	}{
		{
			name:         "test_content_type",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":"kaede"}`,
			expectedBody: NewFailureResponse(http.StatusUnsupportedMediaType, "The 'Content-Type' header is not 'application/json'!"),
			withHeader:   false,
		},
		{
			name:         "test_json_bad_format",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":"kaede",examplebadinput}`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains a badly formatted JSON at position 40!"),
			withHeader:   true,
		},
		{
			name:         "test_json_unexpected_eof",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":"kaede"`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains a badly-formed JSON!"),
			withHeader:   true,
		},
		{
			name:         "test_json_wrong_data_type",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":1234}`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains an invalid value for the \"password\" field at position 35!"),
			withHeader:   true,
		},
		{
			name:         "test_json_unknown_field",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":"kaede","unknownAttribute":"1234"}`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains unknown field '\"unknownAttribute\"'!"),
			withHeader:   true,
		},
		{
			name:         "test_json_empty",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        "",
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body must not be empty!"),
			withHeader:   true,
		},
		{
			name:         "test_json_too_large",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede"}`,
			expectedBody: NewFailureResponse(http.StatusRequestEntityTooLarge, "Request body must not be larger than 512 bytes!"),
			withHeader:   true,
		},
		{
			name:         "test_double_json",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        "{}{}",
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body must only contain a single JSON object!"),
			withHeader:   true,
		},
		{
			name:         "test_success",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kaede","password":"kaede"}`,
			expectedBody: nil,
			withHeader:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.input))
			w := httptest.NewRecorder()
			if tt.withHeader {
				r.Header.Set("Content-Type", "application/json")
			}

			failureResponse := decodeJSONBody(w, r, &AuthRequestBody{})
			assert.JSONEq(t, structToJSON(tt.expectedBody), structToJSON(failureResponse))
		})
	}
}

func TestAuthenticationHandler(t *testing.T) {
	rdb := initializeTestRedis()
	handler := Configure(rdb)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	failureTests := []struct {
		name         string
		method       string
		route        string
		input        string
		expectedBody *FailureResponse
	}{
		{
			name:         "test_wrong_username_and_password",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        `{"username":"kimura","password":"kaori"}`,
			expectedBody: NewFailureResponse(http.StatusUnauthorized, "Username or password do not match!"),
		},
		{
			name:         "test_bad_json",
			method:       http.MethodPost,
			route:        "/api/v1/auth/login",
			input:        "{}{}",
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body must only contain a single JSON object!"),
		},
	}

	successTests := []struct {
		name           string
		method         string
		route          string
		input          string
		expectedStatus int
	}{
		{
			name:           "test_success_login",
			method:         http.MethodPost,
			route:          "/api/v1/auth/login",
			input:          `{"username":"kaede","password":"kaede"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.input))
			w := httptest.NewRecorder()
			r.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(w, r)

			assert.JSONEq(t, structToJSON(tt.expectedBody), w.Body.String())
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.input))
			w := httptest.NewRecorder()
			r.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVerifyHandler(t *testing.T) {
	rdb := initializeTestRedis()
	handler := Configure(rdb)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	testSharedSecret := base32.StdEncoding.EncodeToString([]byte("kaedeKIMURA"))
	defaultOTP, err := totp.GenerateCodeCustom(testSharedSecret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    10,
		Algorithm: otp.AlgorithmSHA512,
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	failureTests := []struct {
		name           string
		input          string
		expectedBody   *FailureResponse
		expectedStatus int
		withHeader     bool
	}{
		{
			name:           "test_without_header",
			input:          "{}",
			expectedBody:   NewFailureResponse(http.StatusUnauthorized, "Please provide an 'Authorization' header!"),
			expectedStatus: http.StatusUnauthorized,
			withHeader:     false,
		},
	}

	successTests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{
			name:           "test_success_verify",
			username:       "kaede",
			password:       defaultOTP,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verification", strings.NewReader(tt.input))
			w := httptest.NewRecorder()
			if tt.withHeader {
				r.Header.Set("Authorization", "XXX")
			}
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, structToJSON(tt.expectedBody), w.Body.String())
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verification", nil)
			w := httptest.NewRecorder()
			r.SetBasicAuth(tt.username, tt.password)
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
