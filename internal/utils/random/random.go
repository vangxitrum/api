package random

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"
)

const letterBytes = "012345679ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func GenerateRandomString(n int) string {
	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	noPaddingEncoder := base64.StdEncoding.WithPadding(base64.NoPadding)

	return noPaddingEncoder.EncodeToString(b)
}

func GenerateRandomCode(n int) string {
	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return fmt.Sprintf("%0*d", n, rand.Intn(900000)+100000)
}

func TruncateString(s string, l int) string {
	if len(s) <= 2*l {
		return s
	}

	return s[:l] + "..." + s[len(s)-l:]
}
