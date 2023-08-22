package blockchain

import (
	"bytes"
	"fmt"
	"golang-blockchain/wallet"
)

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

func (txin *TxInput) UsesKey(pubkeyHash []byte) bool {
	lockHash := wallet.PublicKeyHash(txin.PubKey)
	return bytes.Equal(lockHash, pubkeyHash)
}

func (txout *TxOutput) Lock(address []byte) {
	publicKeyHash := wallet.Base58Decode(address)
	checksum := len(publicKeyHash) - 4
	publicKeyHash = publicKeyHash[1:checksum]
	txout.PubKeyHash = publicKeyHash
}

func (txout *TxOutput) IsLocked(pubKeyHash []byte) bool {
	fmt.Printf("txout.PubKeyHash: %x\n", txout.PubKeyHash)
	fmt.Printf("pubKeyHash:       %x\n", pubKeyHash)

	return bytes.Equal(txout.PubKeyHash, pubKeyHash)
}

func NewTxOutput(value int, address string) *TxOutput {
	txo := &TxOutput{
		Value:      value,
		PubKeyHash: nil,
	}
	txo.Lock([]byte(address))
	return txo
}
