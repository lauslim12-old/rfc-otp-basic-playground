// In this package, we are using 'redismock' with proper mocks as we are expecting the Redis to function properly.
package session

import (
	"errors"
	"fmt"
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
	sessionID, err := GenerateSessionID(32)
	if err != nil {
		log.Fatal(err.Error())
	}
	sessionKey := fmt.Sprintf("sess:%s", sessionID)

	t.Run("test_set_key_success", func(t *testing.T) {
		mock.ExpectSet(sessionKey, "randomUser", sessionExpiration).SetVal("")

		err := service.Set(sessionID, "randomUser")
		assert.Equal(t, nil, err)
	})

	t.Run("test_set_key_failure", func(t *testing.T) {
		mock.ExpectSet(sessionKey, "randomUser", sessionExpiration).SetErr(errors.New("Expect an error!"))

		err := service.Set(sessionID, "randomUser")
		assert.NotNil(t, err)
	})
}

func TestGet(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	service := New(rdb, sessionExpiration)
	sessionID, err := GenerateSessionID(32)
	if err != nil {
		log.Fatal(err.Error())
	}
	sessionKey := fmt.Sprintf("sess:%s", sessionID)

	t.Run("test_get_key_success", func(t *testing.T) {
		mock.ExpectGet(sessionKey).SetVal("randomUser")

		res, err := service.Get(sessionID)
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Equal(t, "randomUser", res)
	})

	t.Run("test_get_key_fail_nil", func(t *testing.T) {
		mock.ExpectGet(sessionKey).RedisNil()

		res, err := service.Get(sessionID)
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Equal(t, "", res)
	})

	t.Run("test_get_key_fail_err", func(t *testing.T) {
		mock.ExpectGet(sessionKey).SetErr(errors.New("Expect an error!"))

		_, err := service.Get(sessionID)
		assert.Equal(t, "Expect an error!", err.Error())
	})
}

func TestAll(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	service := New(rdb, sessionExpiration)

	t.Run("test_get_keys_null", func(t *testing.T) {
		mock.ExpectScan(0, "sess:*", 10).RedisNil()

		res, err := service.All()
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Equal(t, []KeyAndUser(nil), res)
	})

	t.Run("test_get_keys_success", func(t *testing.T) {
		expectedOutput := []KeyAndUser{{"sess:1", "mock-user"}, {"sess:2", "mock-user"}}
		mockSessions := []string{"sess:1", "sess:2"}

		mock.ExpectScan(0, "sess:*", 10).SetVal(mockSessions, 0)
		mock.ExpectGet("sess:1").SetVal("mock-user")
		mock.ExpectGet("sess:2").SetVal("mock-user")

		res, err := service.All()
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Equal(t, expectedOutput, res)
	})
}

func TestBlacklistOTP(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	service := New(rdb, sessionExpiration)

	t.Run("test_blacklist_otp_success", func(t *testing.T) {
		mock.ExpectSAdd("blacklisted_otps", "123").SetVal(1)

		err := service.BlacklistOTP("123")
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Nil(t, err)
	})

	t.Run("test_blacklist_otp_fail", func(t *testing.T) {
		mock.ExpectSAdd("blacklisted_otps", "123").SetErr(errors.New("An error!"))

		err := service.BlacklistOTP("123")
		assert.NotNil(t, err)
	})
}

func TestCheckBlacklistOTP(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	service := New(rdb, sessionExpiration)

	t.Run("test_check_blacklist_otp_success", func(t *testing.T) {
		mock.ExpectSIsMember("blacklisted_otps", "123").SetVal(true)

		res, err := service.CheckBlacklistOTP("123")
		if err != nil {
			log.Fatal(err.Error())
		}

		assert.Nil(t, err)
		assert.Equal(t, true, res)
	})

	t.Run("test_check_blacklist_otp_fail", func(t *testing.T) {
		mock.ExpectSIsMember("blacklisted_otps", "123").SetErr(errors.New("An error!"))

		_, err := service.CheckBlacklistOTP("123")
		assert.NotNil(t, err)
	})
}
