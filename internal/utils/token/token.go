package token

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

const (
	signatureSize        = 65 // bytes
	signatureRIRangeBase = 27
)

type TokenIssuer struct {
	accessTokenExpiresIn         time.Duration
	decodedAccessTokenPrivateKey []byte
	decodedAccessTokenPublicKey  []byte

	refreshTokenExpiresIn         time.Duration
	decodedRefreshTokenPrivateKey []byte
	decodedRefreshTokenPublicKey  []byte
}

func NewTokenIssuer(
	accessTokenPrivateKey string,
	accessTokenPublicKey string,
	refreshTokenPrivateKey string,
	refreshTokenPublicKey string,
	accessTokenExpiredIn int,
	refreshTokenExpiredIn int,
) *TokenIssuer {
	decodedAccessTokenPrivateKey, err := base64.StdEncoding.DecodeString(accessTokenPrivateKey)
	if err != nil {
		log.Fatal("Can not initialize token issuer!")
	}

	decodedAccessTokenPublicKey, err := base64.StdEncoding.DecodeString(accessTokenPublicKey)
	if err != nil {
		log.Fatal("Can not initialize token issuer!")
	}

	decodedRefreshTokenPrivateKey, err := base64.StdEncoding.DecodeString(refreshTokenPrivateKey)
	if err != nil {
		log.Fatal("Can not initialize token issuer!")
	}

	decodedRefreshTokenPublicKey, err := base64.StdEncoding.DecodeString(refreshTokenPublicKey)
	if err != nil {
		log.Fatal("Can not initialize token issuer!")
	}

	return &TokenIssuer{
		decodedAccessTokenPrivateKey:  decodedAccessTokenPrivateKey,
		decodedAccessTokenPublicKey:   decodedAccessTokenPublicKey,
		decodedRefreshTokenPrivateKey: decodedRefreshTokenPrivateKey,
		decodedRefreshTokenPublicKey:  decodedRefreshTokenPublicKey,
		accessTokenExpiresIn:          time.Duration(accessTokenExpiredIn) * time.Second,
		refreshTokenExpiresIn:         time.Duration(refreshTokenExpiredIn) * time.Second,
	}
}

func (t TokenIssuer) createToken(
	payload map[string]string,
	decodePrivateKey []byte,
	expiresIn time.Duration,
) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(decodePrivateKey)
	if err != nil {
		return "", fmt.Errorf("create token: parse key: %w", err)
	}

	now := time.Now().UTC()
	claims := make(jwt.MapClaims)
	claims["sub"] = payload
	claims["exp"] = now.Add(expiresIn).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		return "", fmt.Errorf("create sign token: %w", err)
	}

	return token, nil
}

func (t TokenIssuer) CreateCredential(userId uuid.UUID) (string, string, float64, error) {
	accessToken, err := t.createToken(
		map[string]string{"user_id": userId.String(), "role": "user"},
		t.decodedAccessTokenPrivateKey,
		t.accessTokenExpiresIn,
	)
	if err != nil {
		return "", "", 0, fmt.Errorf("create access token: %w", err)
	}

	refreshToken, err := t.createToken(
		map[string]string{"user_id": userId.String(), "role": "user"},
		t.decodedRefreshTokenPrivateKey,
		t.refreshTokenExpiresIn,
	)
	if err != nil {
		return "", "", 0, fmt.Errorf("create refresh token: %w", err)
	}

	return accessToken, refreshToken, t.accessTokenExpiresIn.Seconds(), nil
}

func (t TokenIssuer) ValidateAccessToken(token string) (map[string]interface{}, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(t.decodedAccessTokenPublicKey)
	if err != nil {
		return nil, fmt.Errorf("validate token: parse key: %w", err)
	}

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("validate token: invalid token")
	}

	exp, err := strconv.ParseFloat(fmt.Sprint(claims["exp"]), 64)
	if err != nil {
		return nil, fmt.Errorf("validate access token: parse exp: %w", err)
	}

	if time.Unix(int64(exp), 0).Before(time.Now().UTC()) {
		return nil, fmt.Errorf("validate access token: token expired")
	}

	return claims["sub"].(map[string]interface{}), nil
}

func (t TokenIssuer) ValidateRefreshToken(token string) (map[string]interface{}, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(t.decodedRefreshTokenPublicKey)
	if err != nil {
		return nil, fmt.Errorf("validate: parse key: %w", err)
	}

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("validate: invalid token")
	}

	exp, err := strconv.ParseFloat(fmt.Sprint(claims["exp"]), 64)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token: parse exp: %w", err)
	}

	if time.Unix(int64(exp), 0).Before(time.Now().UTC()) {
		return nil, fmt.Errorf("validate refresh token: token expired")
	}

	return claims["sub"].(map[string]interface{}), nil
}
