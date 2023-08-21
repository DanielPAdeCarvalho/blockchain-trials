package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	saldo, validOutputs := chain.FindSpendableOutputs(from, amount)
	if saldo < amount {
		log.Printf("Error: not enough funds for this transaction")
	}

	for id, outputs := range validOutputs {
		id, err := hex.DecodeString(id)
		Handle(err)

		for _, output := range outputs {
			input := TxInput{
				ID:  id,
				Out: output,
				Sig: from,
			}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TxOutput{
		Value:  amount,
		PubKey: to,
	})
	if saldo > amount {
		outputs = append(outputs, TxOutput{
			Value:  saldo - amount,
			PubKey: from,
		})
	}
	transaction := Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}
	transaction.SetID()

	return &transaction
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{[]byte{}, -1, data}
	txout := TxOutput{100, to}
	tx := Transaction{nil, []TxInput{txin}, []TxOutput{txout}}
	tx.SetID()
	return &tx
}

func (tx *Transaction) IsCoinbase() bool {
	if len(tx.Inputs) == 1 &&
		len(tx.Inputs[0].ID) == 0 &&
		tx.Inputs[0].Out == -1 {
		return true
	}
	return false
}
