package utils

import (
	"log"

	"github.com/mr-tron/base58"
)

func Base58Encode(input []byte) []byte {
	encoding := base58.Encode(input)
	return []byte(encoding)
}

func Base58Decode(input []byte) []byte {
	decoded, err := base58.Decode(string(input))
	if err != nil {
		log.Fatal(err)
	}
	return decoded
}
