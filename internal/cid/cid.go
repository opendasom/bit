// Package cid converts CIDv0 strings and their sha2-256 digests.
package cid

import (
	"errors"
	"fmt"
	"math/big"
)

const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var alphabetIndex [256]int

func init() {
	for i := range alphabetIndex {
		alphabetIndex[i] = -1
	}
	for i, b := range []byte(alphabet) {
		alphabetIndex[b] = i
	}
}

func DigestFromCIDV0(value string) ([32]byte, error) {
	var digest [32]byte
	raw, err := decodeBase58BTC(value)
	if err != nil {
		return digest, err
	}
	if len(raw) != 34 || raw[0] != 0x12 || raw[1] != 0x20 {
		return digest, fmt.Errorf("unsupported CID format: expected CIDv0 sha2-256 multihash")
	}
	copy(digest[:], raw[2:])
	return digest, nil
}

func CIDV0FromDigest(digest [32]byte) string {
	raw := make([]byte, 34)
	raw[0] = 0x12
	raw[1] = 0x20
	copy(raw[2:], digest[:])
	return encodeBase58BTC(raw)
}

func encodeBase58BTC(data []byte) string {
	x := new(big.Int).SetBytes(data)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var out []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		out = append(out, alphabet[mod.Int64()])
	}
	for _, b := range data {
		if b != 0 {
			break
		}
		out = append(out, alphabet[0])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func decodeBase58BTC(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("empty CID")
	}
	x := big.NewInt(0)
	base := big.NewInt(58)
	for i := 0; i < len(value); i++ {
		c := value[i]
		if alphabetIndex[c] < 0 {
			return nil, fmt.Errorf("invalid base58btc character %q", c)
		}
		x.Mul(x, base)
		x.Add(x, big.NewInt(int64(alphabetIndex[c])))
	}
	decoded := x.Bytes()
	leadingZeros := 0
	for leadingZeros < len(value) && value[leadingZeros] == alphabet[0] {
		leadingZeros++
	}
	if leadingZeros > 0 {
		decoded = append(make([]byte, leadingZeros), decoded...)
	}
	return decoded, nil
}
