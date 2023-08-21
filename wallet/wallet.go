package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"golang-blockchain/utils"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func (w Wallet) Address() []byte {
	publicKey := PublicKeyHash(w.PublicKey)
	version := append([]byte{version}, publicKey...)
	checksum := Checksum(version)
	hash := append(version, checksum...)
	address := utils.Base58Encode(hash)
	return address
}

func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curva := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curva, rand.Reader)
	if err != nil {
		panic(err)
	}

	publicKey := append(privateKey.PublicKey.X.Bytes(),
		privateKey.PublicKey.Y.Bytes()...)
	return *privateKey, publicKey
}

func MakeWallet() *Wallet {
	privateKey, publicKey := NewKeyPair()
	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
}

func PublicKeyHash(publicKey []byte) []byte {
	hash := sha256.Sum256(publicKey)
	return hash[:]
}

func Checksum(payload []byte) []byte {
	hash := sha256.Sum256(payload)
	hash = sha256.Sum256(hash[:])

	return hash[:checksumLength]
}
