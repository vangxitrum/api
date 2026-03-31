package wallet

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mdobak/go-xerrors"
)

const (
	SignatureSize        = 65 // bytes
	SignatureRIRangeBase = 27
)

func VerifySignature(walletAddress common.Address, signature string) error {
	if !common.IsHexAddress(walletAddress.Hex()) {
		return xerrors.New("Invalid address")
	}

	if HasHexPrefix(signature) {
		signature = signature[2:]
	}

	if len(signature) != 2*SignatureSize {
		return xerrors.New("Invalid signature size")
	}

	if !IsHex(signature) {
		return xerrors.New("Invalid signature hex")
	}

	var signatureBytes [SignatureSize]byte
	copy(signatureBytes[:], common.FromHex(signature))

	if signatureBytes[SignatureSize-1] < SignatureRIRangeBase {
		signatureBytes[SignatureSize-1] += SignatureRIRangeBase
	}

	return nil
}

func HasHexPrefix(s string) bool {
	return len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X')
}

func IsHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

func IsHex(s string) bool {
	if len(s)%2 != 0 {
		return false
	}

	for _, c := range []byte(s) {
		if !IsHexCharacter(c) {
			return false
		}
	}

	return true
}
