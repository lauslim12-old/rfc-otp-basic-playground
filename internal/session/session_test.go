// In this package, we are using 'redismock' with proper mocks as we are expecting the Redis to function properly.
package session

import (
	"errors"
	"log"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
)

// Default is 15 minutes for the cache.
var sessionExpiration = time.Minute * 15

func TestSet(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	service := New(rdb, sessionExpiration)
	sessionKey, err := GenerateSessionID(32)
	if err != nil {
		log.Fatal(err.Error())
	}

	t.Run("test_set_key_success", func(t *testing.T) {
		mock.ExpectSet(sessionKey, "randomUser", sessionExpiration).SetVal("")

		err := service.Set(sessionKey, "randomUser")
		assert.Equal(t, nil, err)
	})

	t.Run("test_set_key_failure", func(t *testing.T) {
		mock.ExpectSet(sessionKey, "randomUser", sessionExpiration).SetErr(errors.New("Expect an error!"))

		err := service.Set(sessionKey, "randomUser")
		assert.NotNil(t, err)
	})
}

func TestGet(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	service := New(rdb, sessionExpiration)
	sessionKey, err := GenerateSessionID(32)
	if err != nil {
		log.Fatal(err.Error())
	}

	t.Run("test_get_key_success", func(t *testing.T) {
		mock.ExpectGet(sessionKey).SetVal("randomUser")

		res, err := service.Get(sessionKey)
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Equal(t, "randomUser", res)
	})

	t.Run("test_get_key_fail_nil", func(t *testing.T) {
		mock.ExpectGet(sessionKey).RedisNil()

		res, err := service.Get(sessionKey)
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Equal(t, "", res)
	})

	t.Run("test_get_key_fail_err", func(t *testing.T) {
		mock.ExpectGet(sessionKey).SetErr(errors.New("Expect an error!"))

		_, err := service.Get(sessionKey)
		assert.Equal(t, "Expect an error!", err.Error())
	})
}
