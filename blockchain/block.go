package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type Block struct {
	Hash        []byte
	Transaction []*Transaction
	PrevHash    []byte
	Nonce       int
}

func CreateBlock(trans []*Transaction, prevHash []byte) *Block {
	block := &Block{
		[]byte{},
		trans,
		prevHash,
		0,
	}
	pow := NewProof(block)
	nonce, hash := pow.Run()
	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func Genesis(coinbase *Transaction) *Block {
	return CreateBlock(
		[]*Transaction{coinbase},
		[]byte{},
	)
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encod := gob.NewEncoder(&result)
	err := encod.Encode(b)
	Handle(err)

	return result.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block
	decod := gob.NewDecoder(bytes.NewReader(data))
	err := decod.Decode(&block)
	Handle(err)

	return &block
}

func (b *Block) HashTransaction() []byte {
	var txHashes [][]byte
	var txHashe [32]byte
	for _, tx := range b.Transaction {
		txHashes = append(txHashes, tx.ID)
	}
	txHashe = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHashe[:]
}

func Handle(err error) {
	if err != nil {
		panic(err)
	}
}
