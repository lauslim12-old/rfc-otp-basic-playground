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
	period := 30
	digits := 10
	currentTime := time.Now().Unix()
	counter := currentTime / int64(period)
	hashFunction := sha512.New

	// Create OTP.
	token, err := otp.Generate(counter, digits, sharedSecret, hashFunction)
	if err != nil {
		panic(err)
	}

	// Get my OTP.
	success := fmt.Sprintf(
		"My OTP is '%s', with shared secret '%s', with the counter being '%d'.", *token, sharedSecret, counter,
	)
	fmt.Println(success)
}
