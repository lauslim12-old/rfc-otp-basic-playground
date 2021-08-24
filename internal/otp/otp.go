package otp

import (
	"crypto/hmac"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"strings"
)

// TOTPConfig in order to act as a baseline of TOTP configurations.
type TOTPConfig struct {
	Secret    string           // OTP shared secret.
	Period    int64            // Period of the token will be validated.
	Timestamp int64            // Timestamp or current time in UNIX time.
	Digits    int              // Digits requested for the OTP.
	Hasher    func() hash.Hash // Hash algorithm for the OTP.
}

// TOTPValidateConfig to configure validation parameters.
type TOTPValidateConfig struct {
	Secret    string           // OTP shared secret.
	Period    int64            // Period of the token will be validated.
	Timestamp int64            // Timestamp or current time in UNIX time.
	Digits    int              // Digits requested for the OTP.
	Hasher    func() hash.Hash // Hash algorithm for the OTP.
	Window    int64            // How long in a timeframe should an OTP be tolerated.
}

// This function is an utility function to convert a secret (base32 encoded) into byte form.
func transformSecret(sharedSecret string) ([]byte, error) {
	// Transform into bytes.
	byteString, err := base32.StdEncoding.DecodeString(sharedSecret)
	if err != nil {
		return nil, err
	}

	// Returns our transformed secret.
	return byteString, nil
}

// This function is an utility function to convert an integer into byte form.
func transformCounter(counter int64) []byte {
	// Transform into bytes.
	counterInBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterInBytes, uint64(counter))

	// Returns our transformed counter.
	return counterInBytes
}

// This function will pad an OTP with leading zeroes shall the digits are not enough.
func pad(otp, digits int) string {
	return fmt.Sprintf(fmt.Sprintf("%%0%dd", digits), otp)
}

// This function will validate a TOTP using constant time compare.
// Window is used as the interval - the window of counter values to test.
func Verify(otp string, options TOTPValidateConfig) (bool, error) {
	// Remove whitespaces from the passed OTP and calculate counter.
	passcode := strings.TrimSpace(otp)
	counter := options.Timestamp / options.Period

	// Check if the length of the OTP is not equal to specified digits.
	if len(passcode) != options.Digits {
		return false, errors.New("passcode is not equal to the specified digits in length")
	}

	// We will try to safely compare two strings at a single moment.
	// Also try to generate tokens in allowed windows. If one match, then that token is valid.
	for i := counter - options.Window; i <= counter+options.Window; i++ {
		generatedToken, err := Generate(TOTPConfig{
			Secret:    options.Secret,
			Period:    options.Period,
			Timestamp: options.Timestamp,
			Digits:    options.Digits,
			Hasher:    options.Hasher,
		})
		if err != nil {
			return false, err
		}

		if subtle.ConstantTimeCompare([]byte(passcode), []byte(generatedToken)) == 1 {
			return true, nil
		}
	}

	return false, nil
}

// This function will generate a new OTP. In this case, it's TOTP.
// Reference: https://datatracker.ietf.org/doc/html/rfc6238.
func Generate(options TOTPConfig) (string, error) {
	// Calculate counters.
	counter := options.Timestamp / options.Period

	// Removes whitespaces for some secrets.
	// Transform to uppercase to conform to the RFC.
	secretTrimmed := strings.TrimSpace(options.Secret)
	secretTrimmed = strings.ToUpper(secretTrimmed)

	// Transform 'counter' into a byte array.
	counterInBytes := transformCounter(counter)

	// Transform 'secret' into a byte array.
	secretInBytes, err := transformSecret(secretTrimmed)
	if err != nil {
		return "", err
	}

	// Create a new OTP token based on the inputs.
	hmac := hmac.New(options.Hasher, secretInBytes)
	hmac.Write(counterInBytes)
	digest := hmac.Sum(nil)

	// After getting the digest, we get the properties of the OTP.
	// Everything has to be casted to integer to round them.
	offset := int(digest[len(digest)-1] & 15)
	otp := ((int(digest[offset] & 127)) << 24) |
		((int(digest[offset+1] & 255)) << 16) |
		((int(digest[offset+2] & 255)) << 8) |
		(int(digest[offset+3] & 255))
	otp = otp % int(math.Pow10(options.Digits))

	// Pad the OTP with leading zeroes.
	token := pad(otp, options.Digits)

	// Return the newly created OTP.
	return token, nil
}
