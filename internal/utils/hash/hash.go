package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashString(input string) (string, error) {
	hashedInput, err := bcrypt.GenerateFromPassword(
		[]byte(input),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", fmt.Errorf(
			"could not hash %w",
			err,
		)
	}
	return string(hashedInput), nil
}

func HashStringSHA256(input string) (
	string, error,
) {
	hasher := sha256.New()
	_, err := hasher.Write([]byte(input))
	if err != nil {
		return "", fmt.Errorf(
			"could not hash email %w",
			err,
		)
	}
	hashedInput := hasher.Sum(nil)
	return hex.EncodeToString(hashedInput), nil
}
