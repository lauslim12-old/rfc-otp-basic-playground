package otp

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"hash"
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
}

func TestGenerate(t *testing.T) {
	failureTests := []struct {
		name    string
		counter int64
		digits  int
		secret  string
		hasher  func() hash.Hash
	}{
		{
			name:    "test_negative_input",
			counter: -1,
			digits:  6,
			secret:  toBase32("The quick brown fox jumps over the lazy dog."),
			hasher:  sha512.New,
		},
		{
			name:    "test_invalid_base32",
			counter: 123,
			digits:  6,
			secret:  "invalid_base32",
			hasher:  sha512.New,
		},
	}

	successTests := []struct {
		name           string
		counter        int64
		digits         int
		secret         string
		hasher         func() hash.Hash
		expectedOutput string
	}{
		{
			name:           "test_otp_1_10_digits",
			counter:        54324343,
			digits:         10,
			secret:         toBase32("The quick brown fox jumps over the lazy dog."),
			hasher:         sha512.New,
			expectedOutput: "0582933009",
		},
		{
			name:           "test_otp_2_6_digits",
			counter:        54324351,
			digits:         6,
			secret:         toBase32("The quick brown fox jumps over the lazy dog."),
			hasher:         sha512.New,
			expectedOutput: "934368",
		},
		{
			name:           "test_otp_3_sha256_hasher",
			counter:        54324354,
			digits:         6,
			secret:         toBase32("The quick brown fox jumps over the lazy dog."),
			hasher:         sha256.New,
			expectedOutput: "181011",
		},
		{
			name:           "test_otp_4_60_period",
			counter:        27162206,
			digits:         10,
			secret:         toBase32("The quick brown fox jumps over the lazy dog."),
			hasher:         sha512.New,
			expectedOutput: "1796746380",
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := Generate(tt.counter, tt.digits, tt.secret, tt.hasher)
			if err == nil {
				t.Errorf("Test-cases should return error(s)! Got: %v!", otp)
			}
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := Generate(tt.counter, tt.digits, tt.secret, tt.hasher)
			if err != nil {
				t.Errorf("Test-cases should not return error(s)! Got: %v, expected: %v!", err, tt.expectedOutput)
			}

			if *otp != tt.expectedOutput {
				t.Errorf("OTP and the expected output are not the same! Got: %v, expected: %v!", *otp, tt.expectedOutput)
			}
		})
	}
}
