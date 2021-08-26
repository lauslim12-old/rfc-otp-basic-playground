package application

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/lauslim12/fullstack-otp/internal/session"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// SuccessResponse is used to handle successful requests.
type SuccessResponse struct {
	Status  string      `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse is used to create a default, new success response.
func NewSuccessResponse(code int, message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Status:  "success",
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// FailureResponse is used to handle failed requests.
type FailureResponse struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewFailureResponse is used to create a default, new failure response.
func NewFailureResponse(code int, message string) *FailureResponse {
	return &FailureResponse{
		Status:  "fail",
		Code:    code,
		Message: message,
	}
}

// AuthRequestBody is to create the basic type of an incoming authentication request body.
type AuthRequestBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ContextKey is used to pass around userID in requests.
type ContextKey struct{}

// Utility function to send succesful response.
func sendSuccessResponse(w http.ResponseWriter, successResponse *SuccessResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(successResponse.Code)
	json.NewEncoder(w).Encode(successResponse)
}

// Utility function to send failure response.
func sendFailureResponse(w http.ResponseWriter, failureResponse *FailureResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(failureResponse.Code)
	json.NewEncoder(w).Encode(failureResponse)
}

// Utility function to decode a JSON request body.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) *FailureResponse {
	// Check if Header is 'Content-Type: application/json'.
	if r.Header.Get("Content-Type") != "application/json" {
		return NewFailureResponse(http.StatusUnsupportedMediaType, "The 'Content-Type' header is not 'application/json'!")
	}

	// Parse body, and set max bytes reader (512 bytes).
	r.Body = http.MaxBytesReader(w, r.Body, 512)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		// Handle syntax errors.
		case errors.As(err, &syntaxError):
			errorMessage := fmt.Sprintf("Request body contains a badly formatted JSON at position %d!", syntaxError.Offset)
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle unexpected EOFs.
		case errors.Is(err, io.ErrUnexpectedEOF):
			errorMessage := "Request body contains a badly-formed JSON!"
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle wrong data-type in request body.
		case errors.As(err, &unmarshalTypeError):
			errorMessage := fmt.Sprintf("Request body contains an invalid value for the %q field at position %d!", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle unknown fields.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			errorMessage := fmt.Sprintf("Request body contains unknown field '%s'!", fieldName)
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle empty request body.
		case errors.Is(err, io.EOF):
			errorMessage := "Request body must not be empty!"
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle too large body.
		case err.Error() == "http: request body too large":
			errorMessage := "Request body must not be larger than 512 bytes!"
			return NewFailureResponse(http.StatusRequestEntityTooLarge, errorMessage)

		// Handle other errors.
		default:
			return NewFailureResponse(http.StatusInternalServerError, err.Error())
		}
	}
	defer r.Body.Close()

	// Handle if client tries to send more than one JSON object.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		errorMessage := "Request body must only contain a single JSON object!"
		return NewFailureResponse(http.StatusBadRequest, errorMessage)
	}

	// If everything goes well, don't return anything.
	return nil
}

// Configure is used to configure the application (server is initialized in 'main').
func Configure(rdb *redis.Client) http.Handler {
	// Create a Chi instance.
	r := chi.NewRouter()

	// Set up Chi's natural middlewares.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set up custom middleware.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Application-Name", "Fullstack OTP")
			w.Header().Add("Server", "net/http")
			next.ServeHTTP(w, r)
		})
	})

	// Group routes.
	r.Route("/api/v1", func(r chi.Router) {
		// Sample GET route.
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			res := NewSuccessResponse(http.StatusOK, "Welcome to 'net/http' API!", nil)
			sendSuccessResponse(w, res)
		})

		// Subrouter: '/api/v1/auth'.
		r.Route("/auth", func(r chi.Router) {
			// Login route.
			r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
				authRequestBody := &AuthRequestBody{}
				failureResponse := decodeJSONBody(w, r, authRequestBody)
				if failureResponse != nil {
					sendFailureResponse(w, failureResponse)
					return
				}

				// Calculate SHA256 hash to prevent 'ConstantTimeCompare' leaking the length of passwords / usernames.
				// SHA256 is used to quickly generate and verify the hashes - SHA512 would take a bit longer.
				usernameHash := sha256.Sum256([]byte(authRequestBody.Username))
				passwordHash := sha256.Sum256([]byte(authRequestBody.Password))
				expectedUsernameHash := sha256.Sum256([]byte("kaede"))
				expectedPasswordHash := sha256.Sum256([]byte("kaede"))

				// Compare if username and passwords match.
				// Let's claim that the username and password are 'OTP_EXPECTED_USERNAME/PASSWORD' for now.
				usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
				passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1
				if !usernameMatch || !passwordMatch {
					sendFailureResponse(w, NewFailureResponse(http.StatusUnauthorized, "Username or password do not match!"))
					return
				}

				// After this, we should check Redis and verify if there is a cache with this user.
				// If not, simply send them an OTP. The secret, same as above, is 'kaedeKIMURA' for now.
				// 'KIMURA' is the shared secret, 'kaede' is the username. We concatenate them together.
				sharedSecret := base32.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", authRequestBody.Username, "KIMURA")))
				otp, err := totp.GenerateCodeCustom(sharedSecret, time.Now(), totp.ValidateOpts{
					Period:    30,
					Skew:      1,
					Digits:    10,
					Algorithm: otp.AlgorithmSHA512,
				})
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusBadRequest, err.Error()))
					return
				}

				// Make a response body. This is for development only. Production will send the OTP via other methods.
				basicAuthInformation := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", authRequestBody.Username, otp)))
				basicAuthContent := fmt.Sprintf("%s%s", "Basic ", basicAuthInformation)
				decodedBasicAuth, err := base64.StdEncoding.DecodeString(basicAuthInformation)
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
					return
				}

				// Anonymous struct.
				responseData := struct {
					OTP              string `json:"otp"`
					Username         string `json:"user"`
					BasicAuthContent string `json:"basicAuth"`
					DecodedBasicAuth string `json:"decodedBasic"`
					SharedSecret     string `json:"sharedSecret"`
					LoginTime        int64  `json:"loginTime"`
				}{
					OTP:              otp,
					Username:         authRequestBody.Username,
					BasicAuthContent: basicAuthContent,
					DecodedBasicAuth: string(decodedBasicAuth),
					SharedSecret:     sharedSecret,
					LoginTime:        time.Now().Unix(),
				}
				sendSuccessResponse(w, NewSuccessResponse(http.StatusOK, "Sucessfully logged in!", responseData))
			})

			// Verification route.
			r.Post("/verification", func(w http.ResponseWriter, r *http.Request) {
				// Get the Authorization Header.
				username, password, ok := r.BasicAuth()
				if !ok {
					w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
					sendFailureResponse(w, NewFailureResponse(http.StatusUnauthorized, "Please provide an 'Authorization' header!"))
					return
				}

				// Calculate SHA256 of the 'username'.
				usernameHash := sha256.Sum256([]byte(username))
				expectedUsernameHash := sha256.Sum256([]byte("kaede"))

				// Verify OTP.
				sharedSecret := base32.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%sKIMURA", username)))
				validOTP, err := totp.ValidateCustom(password, sharedSecret, time.Now(), totp.ValidateOpts{
					Period:    30,
					Skew:      1,
					Digits:    10,
					Algorithm: otp.AlgorithmSHA512,
				})
				if err != nil && err == otp.ErrValidateInputInvalidLength {
					sendFailureResponse(w, NewFailureResponse(http.StatusBadRequest, "Your OTP does not conform to the length requirements of the validation server!"))
					return
				}
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusBadRequest, err.Error()))
					return
				}

				// Check if OTP and username are valid.
				usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
				if !usernameMatch {
					sendFailureResponse(w, NewFailureResponse(http.StatusUnauthorized, "Username does not match with the database!"))
					return
				}
				if !validOTP {
					sendFailureResponse(w, NewFailureResponse(http.StatusUnauthorized, "Invalid token, wrong TOTP code!"))
					return
				}

				// Check if OTP is blacklisted.
				sess := session.New(rdb, time.Minute*15)
				blacklistedOTP, err := sess.CheckBlacklistOTP(password)
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
					return
				}
				if blacklistedOTP {
					sendFailureResponse(w, NewFailureResponse(http.StatusBadRequest, "The OTP that you entered has been used before!"))
					return
				}

				// Blacklist OTP.
				err = sess.BlacklistOTP(password)
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
					return
				}

				// Set user cache.
				sessionKey, err := session.GenerateSessionID(32)
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
					return
				}

				err = sess.Set(sessionKey, username)
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
					return
				}

				// If successful, dump the user data and everything.
				responseData := struct {
					OTP          string `json:"otp"`
					User         string `json:"user"`
					OK           bool   `json:"ok"`
					ValidOTP     bool   `json:"validOTP"`
					SharedSecret string `json:"sharedSecret"`
					SessionKey   string `json:"sessionKey"`
					VerifyTime   int64  `json:"verifyTime"`
				}{
					OTP:          password,
					User:         username,
					OK:           ok,
					ValidOTP:     validOTP,
					SharedSecret: sharedSecret,
					SessionKey:   sessionKey,
					VerifyTime:   time.Now().Unix(),
				}

				// Send back response.
				http.SetCookie(w, &http.Cookie{
					Name:     "sess",
					Value:    sessionKey,
					Path:     "/",
					Expires:  time.Now().Add(15 * time.Minute),
					HttpOnly: true,
				})
				sendSuccessResponse(w, NewSuccessResponse(http.StatusOK, "OTP and user successfully verified!", responseData))
			})
		})

		// Subrouter: '/api/v1/sessions'.
		r.Route("/sessions", func(r chi.Router) {
			sess := session.New(rdb, time.Minute*15)

			// Check authorization in Redis session.
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check session cookie.
					sessionKey, err := r.Cookie("sess")
					if err != nil {
						sendFailureResponse(w, NewFailureResponse(http.StatusBadRequest, "No session found. Please log in again!"))
						return
					}

					// Check if session exists.
					userID, err := sess.Get(sessionKey.Value)
					if userID == "" {
						sendFailureResponse(w, NewFailureResponse(http.StatusBadRequest, "User with your session ID is not found! Please log in again!"))
						return
					}
					if err != nil {
						sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
						return
					}

					// Allow next, pass user ID via context.
					ctx := context.WithValue(r.Context(), ContextKey{}, userID)
					next.ServeHTTP(w, r.Clone(ctx))
				})
			})

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				// Get context and parse the value.
				userID := r.Context().Value(ContextKey{}).(string)
				if userID == "" {
					sendFailureResponse(w, NewFailureResponse(http.StatusUnauthorized, "You are unauthorized to access this route!"))
					return
				}

				// Get all sessions.
				keys, err := sess.All()
				if err != nil {
					sendFailureResponse(w, NewFailureResponse(http.StatusInternalServerError, err.Error()))
					return
				}

				// Make response body.
				resp := struct {
					KeyAndUsers interface{} `json:"keys"`
					UserID      string      `json:"user"`
				}{
					KeyAndUsers: keys,
					UserID:      userID,
				}
				sendSuccessResponse(w, NewSuccessResponse(http.StatusOK, "All of the sessions in the application.", resp))
			})
		})

		// Declare method not allowed as a fallback.
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			errorMessage := fmt.Sprintf("Method '%s' is not allowed in this route!", r.Method)
			res := NewFailureResponse(http.StatusMethodNotAllowed, errorMessage)
			sendFailureResponse(w, res)
		})

		// Declare 404 every time a request reaches here.
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			errorMessage := fmt.Sprintf("Route '%s' with method '%s' does not exist in this server!", r.RequestURI, r.Method)
			res := NewFailureResponse(http.StatusNotFound, errorMessage)
			sendFailureResponse(w, res)
		})
	})

	// Return our configured infrastructure.
	return r
}
