package wallet

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/random"
)

const (
	challengeStringLength = 32
	SigningSig            = "signing"
	VerifyWallet          = "verify"
)

func GetChallenge() string {
	return random.GenerateRandomString(challengeStringLength)
}

func FormatMessageForSigning(userAddress, nonce string) string {
	return fmt.Sprintf(
		"Welcome to AIOZ!\n\nClick to sign in and accept the AIOZ Terms of Service: https://aioz.network/\n\nThis request will not trigger a blockchain transaction or cost any gas fees.\n\nYour authentication status will reset after 24 hours.\n\nWallet address:\n%s\n\nNonce:\n%s\n",
		userAddress,
		nonce,
	)
}

func FormatMessageForVerify(userAddress, email, nonce string) string {
	return fmt.Sprintf(
		"Welcome to AIOZ!\n\nClick to sign in and accept the AIOZ Terms of Service: https://aioz.network/\n\nThis request will not trigger a blockchain transaction or cost any gas fees.\n\nWallet address:\n%s\n\nEmail:\n%s\n\nNonce:\n%s\n",
		userAddress,
		email,
		nonce,
	)
}

func VerifySig(from, sigHex, email, nonce, sigType string) bool {
	return VerifySig1(from, sigHex, email, nonce, sigType) ||
		VerifySig2(from, sigHex, email, nonce, sigType)
}

func VerifySig1(from, sigHex, email, nonce, sigType string) bool {
	fromAddr := common.HexToAddress(from)
	sig := hexutil.MustDecode(sigHex)

	var msg []byte
	switch sigType {
	case SigningSig:
		msg = accounts.TextHash([]byte(FormatMessageForSigning(from, nonce)))
	case VerifyWallet:
		msg = accounts.TextHash([]byte(FormatMessageForVerify(from, email, nonce)))
	default:
		return false
	}

	if sig[64] != 27 && sig[64] != 28 {
		return false
	}
	sig[64] -= 27

	pubKey, err := crypto.SigToPub(msg, sig)
	if err != nil {
		return false
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	return fromAddr == recoveredAddr
}

func VerifySig2(from, sigHex, email, nonce, sigType string) bool {
	fromAddr := common.HexToAddress(from)

	sig := hexutil.MustDecode(sigHex)

	var msg []byte
	switch sigType {
	case SigningSig:
		msg = accounts.TextHash([]byte(FormatMessageForSigning(from, nonce)))
	case VerifyWallet:
		msg = accounts.TextHash([]byte(FormatMessageForVerify(from, email, nonce)))
	default:
		return false
	}
	pubKey, err := crypto.SigToPub(msg, sig)
	if err != nil {
		return false
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	return fromAddr == recoveredAddr
}
