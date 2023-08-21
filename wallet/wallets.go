package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"math/big"
	"os"
)

const walletFile = "./tmp/wallets.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

type SerializableWallet struct {
	PrivateKey []byte
	PublicKey  []byte
}

func (w *Wallet) ToSerializable() SerializableWallet {
	return SerializableWallet{
		PrivateKey: w.PrivateKey.D.Bytes(),
		PublicKey:  w.PublicKey,
	}
}

func FromSerializable(sw SerializableWallet) Wallet {
	priv := new(ecdsa.PrivateKey)
	priv.Curve = elliptic.P256()
	priv.D = new(big.Int).SetBytes(sw.PrivateKey)
	priv.PublicKey.X, priv.PublicKey.Y = priv.Curve.ScalarBaseMult(sw.PrivateKey)
	return Wallet{
		PrivateKey: *priv,
		PublicKey:  sw.PublicKey,
	}
}

func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile()

	return &wallets, err
}

func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet()
	address := fmt.Sprintf("%s", wallet.Address())

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) SaveFile() error {
	serializableWallets := make(map[string]SerializableWallet)
	for address, wallet := range ws.Wallets {
		serializableWallets[address] = wallet.ToSerializable()
	}

	var content bytes.Buffer
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(serializableWallets)
	if err != nil {
		return err
	}

	file, err := os.Create(walletFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content.Bytes())
	if err != nil {
		return err
	}

	return file.Sync()
}

func (ws *Wallets) LoadFile() error {
	file, err := os.Open(walletFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var serializableWallets map[string]SerializableWallet
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&serializableWallets)
	if err != nil {
		return err
	}

	ws.Wallets = make(map[string]*Wallet)
	for address, sw := range serializableWallets {
		wallet := FromSerializable(sw)
		ws.Wallets[address] = &wallet
	}

	return nil
}
