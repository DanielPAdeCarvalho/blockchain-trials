package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"golang-blockchain/wallet"
	"log"
	"math/big"
	"strings"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

func (t *Transaction) Serialize() []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(t)
	if err != nil {
		log.Fatal(err)
	}
	return buffer.Bytes()
}

func (t *Transaction) Hash() []byte {
	var hash [32]byte
	copia := *t
	copia.ID = []byte{}
	hash = sha256.Sum256(copia.Serialize())
	return hash[:]
}

func (t *Transaction) Sign(privateKey ecdsa.PrivateKey, previousTx map[string]Transaction) {
	if t.IsCoinbase() {
		return
	}

	for _, input := range t.Inputs {
		id := hex.EncodeToString(input.ID)
		if previousTx[id].ID == nil {
			log.Fatal("Transaction not found, Previous Transaction is not valid")
		}
	}

	txCopy := t.TrimmedCopy()
	for inputId, input := range t.Inputs {
		id := hex.EncodeToString(input.ID)
		previousTx := previousTx[id]
		txCopy.Inputs[inputId].Signature = nil
		txCopy.Inputs[inputId].PubKey = previousTx.Outputs[input.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inputId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, txCopy.ID)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)
		t.Inputs[inputId].Signature = signature
	}
}

func (t *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, input := range t.Inputs {
		txI := TxInput{
			ID:        input.ID,
			Out:       input.Out,
			PubKey:    nil,
			Signature: nil,
		}
		inputs = append(inputs, txI)
	}

	for _, output := range t.Outputs {
		txO := TxOutput{
			Value:      output.Value,
			PubKeyHash: output.PubKeyHash,
		}
		outputs = append(outputs, txO)
	}
	return Transaction{
		ID:      t.ID,
		Inputs:  inputs,
		Outputs: outputs,
	}
}

func (t *Transaction) Verify(previousTxs map[string]Transaction) bool {
	if t.IsCoinbase() {
		return true
	}

	for _, input := range t.Inputs {
		id := hex.EncodeToString(input.ID)
		if previousTxs[id].ID == nil {
			log.Fatal("Transaction not found, Previous Transaction is not valid")
		}
	}

	txCopy := t.TrimmedCopy()
	curva := elliptic.P256()
	for inputID, input := range t.Inputs {
		id := hex.EncodeToString(input.ID)
		previousTx := previousTxs[id]
		txCopy.Inputs[inputID].Signature = nil
		txCopy.Inputs[inputID].PubKey = previousTx.Outputs[input.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inputID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		signatureLength := len(input.Signature)
		r.SetBytes(input.Signature[:signatureLength/2])
		s.SetBytes(input.Signature[signatureLength/2:])

		x := big.Int{}
		y := big.Int{}
		keyLength := len(input.PubKey)
		x.SetBytes(input.PubKey[:keyLength/2])
		y.SetBytes(input.PubKey[keyLength/2:])

		pubKey := ecdsa.PublicKey{
			Curve: curva,
			X:     &x,
			Y:     &y,
		}
		if !ecdsa.Verify(&pubKey, txCopy.ID, &r, &s) {
			return false
		}
	}
	return true
}
func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	wallets, err := wallet.CreateWallets()
	Handle(err)

	w := wallets.GetWallet(from)
	if w.PrivateKey.D == nil {
		log.Fatal("Wallet not found")
	}
	publicKey := wallet.PublicKeyHash(w.PublicKey)
	saldo, validOutputs := chain.FindSpendableOutputs(publicKey, amount)
	if saldo < amount {
		fmt.Printf("O usuario so tem %d de saldo", saldo)
		log.Panic("Error: not enough funds for this transaction")
	}

	for id, outputs := range validOutputs {
		id, err := hex.DecodeString(id)
		Handle(err)

		for _, output := range outputs {
			input := TxInput{
				ID:        id,
				Out:       output,
				Signature: nil,
				PubKey:    w.PublicKey,
			}
			inputs = append(inputs, input)
		}
	}

	txOut := *NewTxOutput(amount, to)
	outputs = append(outputs, txOut)
	if saldo > amount {
		txOut := *NewTxOutput(saldo-amount, from)
		outputs = append(outputs, txOut)
	}
	transaction := Transaction{
		Inputs:  inputs,
		Outputs: outputs,
	}
	transaction.ID = transaction.Hash()
	chain.SignTransaction(&transaction, w.PrivateKey)

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

	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTxOutput(100, to)
	tx := Transaction{
		nil,
		[]TxInput{txin},
		[]TxOutput{*txout},
	}
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

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       PubKeyHash: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
