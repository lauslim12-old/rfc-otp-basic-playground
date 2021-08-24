package otp

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"testing"
)

func toBase32(str string) string {
	return base32.StdEncoding.EncodeToString([]byte(str))
}

func TestTransformSecret(t *testing.T) {
	t.Run("test_transform_not_base32", func(t *testing.T) {
		input := "not_base32"

		res, err := transformSecret(input)
		if err == nil {
			t.Errorf("Error should be not null! Got: %v!", res)
		}
	})

	t.Run("test_transform_base32", func(t *testing.T) {
		input := toBase32("The quick brown fox jumps over the lazy dog.")

		_, err := transformSecret(input)
		if err != nil {
			t.Errorf("Error should be null! Got: %v!", err)
		}
	})
}

func TestTransformCounter(t *testing.T) {
	t.Run("test_transform_counter", func(t *testing.T) {
		input := 1234

		res := transformCounter(int64(input))
		if res == nil {
			t.Error("The result should be not null!")
		}
	})
}

func TestPad(t *testing.T) {
	t.Run("test_padding", func(t *testing.T) {
		inputOTP, inputDigits := 1234, 6

		res := pad(inputOTP, inputDigits)
		if res != "001234" {
			t.Error("The end result of the padding should be '001234'!")
		}
	})

	t.Run("test_enough_padding", func(t *testing.T) {
		inputOTP, inputDigits := 123456, 6

		res := pad(inputOTP, inputDigits)
		if res != "123456" {
			t.Error("The end result of the padding should be '123456'!")
		}
	})
}

func TestGenerate(t *testing.T) {
	sharedSecret := toBase32("The quick brown fox jumps over the lazy dog.")
	period := 30

	failureTests := []struct {
		name       string
		totpConfig TOTPConfig
	}{
		{
			name: "test_invalid_base32",
			totpConfig: TOTPConfig{
				Secret:    "invalid_base32",
				Period:    int64(period),
				Timestamp: 1629794446,
				Digits:    10,
				Hasher:    sha512.New,
			},
		},
	}

	successTests := []struct {
		name           string
		totpConfig     TOTPConfig
		expectedOutput string
	}{
		{
			name: "test_otp_1_10_digits",
			totpConfig: TOTPConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629794237,
				Digits:    10,
				Hasher:    sha512.New,
			},
			expectedOutput: "2091961511",
		},
		{
			name: "test_otp_2_6_digits",
			totpConfig: TOTPConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629794237,
				Digits:    6,
				Hasher:    sha512.New,
			},
			expectedOutput: "961511",
		},
		{
			name: "test_otp_3_sha256_hasher",
			totpConfig: TOTPConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629794392,
				Digits:    10,
				Hasher:    sha256.New,
			},
			expectedOutput: "0097957743",
		},
		{
			name: "test_otp_4_60_period_sha256",
			totpConfig: TOTPConfig{
				Secret:    sharedSecret,
				Period:    60,
				Timestamp: 1629794446,
				Digits:    10,
				Hasher:    sha256.New,
			},
			expectedOutput: "0565820944",
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := Generate(tt.totpConfig)
			if err == nil {
				t.Errorf("Test-cases should return error(s)! Got: %v!", otp)
			}
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := Generate(tt.totpConfig)
			if err != nil {
				t.Errorf("Test-cases should not return error(s)! Got: %v, expected: %v!", err, tt.expectedOutput)
			}

			if otp != tt.expectedOutput {
				t.Errorf("OTP and the expected output are not the same! Got: %v, expected: %v!", otp, tt.expectedOutput)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	sharedSecret := toBase32("The quick brown fox jumps over the lazy dog.")
	period := 30

	errorTests := []struct {
		name           string
		otp            string
		totpValidation TOTPValidateConfig
	}{
		{
			name: "test_otp_not_enough_length",
			otp:  "1234",
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629780615,
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_invalid_secret_base32",
			otp:  "000000",
			totpValidation: TOTPValidateConfig{
				Secret:    "invalid_base32",
				Period:    int64(period),
				Timestamp: 1629780615,
				Digits:    6,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
	}

	invalidTests := []struct {
		name           string
		otp            string
		totpValidation TOTPValidateConfig
	}{
		{
			name: "test_otp_invalid_because_more_than_30_seconds",
			otp:  "1736605286", // OTP generated at 1629787611.
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629787651, // sometimes, it can be valid for a bit more than 30 seconds.
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_all_zeroes",
			otp:  "000000",
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629787651, // taken from the above.
				Digits:    6,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_before_timestamp",
			otp:  "1736605286", // OTP generated at 1629787611.
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629787555, // 56 seconds before OTP generation time.
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
	}

	successTests := []struct {
		name           string
		otp            string
		totpValidation TOTPValidateConfig
	}{
		{
			name: "test_otp_5_seconds",
			otp:  "2053730166", // OTP generated at 1629795960
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629795965,
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_10_seconds",
			otp:  "2053730166", // OTP generated at 1629795960
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629795970,
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_15_seconds",
			otp:  "2053730166", // OTP generated at 1629795960
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629795975,
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_20_seconds",
			otp:  "2053730166", // OTP generated at 1629795960
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629795980,
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_25_seconds",
			otp:  "2053730166", // OTP generated at 1629795960
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629795985,
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
		{
			name: "test_otp_30_seconds",
			otp:  "2053730166", // OTP generated at 1629795960
			totpValidation: TOTPValidateConfig{
				Secret:    sharedSecret,
				Period:    int64(period),
				Timestamp: 1629795989, // 1 second before 30 seconds
				Digits:    10,
				Hasher:    sha512.New,
				Window:    1,
			},
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := Verify(tt.otp, tt.totpValidation)
			if err == nil {
				t.Errorf("Test case should return an error! Got: %v, expected: %v!", err, valid)
			}
		})
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := Verify(tt.otp, tt.totpValidation)
			if err != nil {
				t.Errorf("Verification test-cases should not return error(s)! Got: %v!", err)
			}

			if valid {
				t.Errorf("Result of the test-cases should be invalid. Got: %v!", valid)
			}
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := Verify(tt.otp, tt.totpValidation)
			if err != nil {
				t.Errorf("Verification test-cases should not return error(s)! Got: %v!", err)
			}

			if !valid {
				t.Errorf("Result of the test-cases should be valid. Got: %v!", valid)
			}
		})
	}
}
