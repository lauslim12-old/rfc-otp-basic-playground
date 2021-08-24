package main

import (
	"crypto/sha512"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/lauslim12/fullstack-otp/internal/otp"
)

func main() {
	// OTP metrics.
	// Reference: https://datatracker.ietf.org/doc/html/rfc6238.
	sharedSecret := base32.StdEncoding.EncodeToString([]byte("The quick brown fox jumps over the lazy dog."))
	totpConfig := otp.TOTPConfig{
		Secret:    sharedSecret,
		Period:    30,
		Digits:    10,
		Timestamp: time.Now().Unix(),
		Hasher:    sha512.New,
	}

	// Create OTP.
	token, err := otp.Generate(totpConfig)
	if err != nil {
		panic(err)
	}

	// Get my OTP.
	success := fmt.Sprintf(
		"My OTP is '%s', with shared secret '%s', with current time being '%d'.", token, sharedSecret, totpConfig.Timestamp,
	)
	fmt.Println(success)

	// Verify my OTP.
	res, err := otp.Verify(token, otp.TOTPValidateConfig{
		Secret:    sharedSecret,
		Period:    30,
		Digits:    10,
		Timestamp: time.Now().Unix(),
		Hasher:    sha512.New,
		Window:    1,
	})
	if err != nil {
		panic(err)
	}

	// Get result.
	validOTP := fmt.Sprintf("My OTP validity is %v.", res)
	fmt.Println(validOTP)
}
