package blockchain

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "Genesis Block Data"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func InitBlockChain(address string) *BlockChain {
	var lastHash []byte
	if DBExists() {
		fmt.Printf("BlockChain already crated, skipping")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		coinbaseTransaction := CoinbaseTx(address, genesisData)
		genesis := Genesis(coinbaseTransaction)
		fmt.Println("Genesis Block created successfully")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err
	})
	Handle(err)
	chain := &BlockChain{LastHash: lastHash, Database: db}
	return chain
}

func ContinueBlockChain(address string) *BlockChain {
	if DBExists() == false {
		fmt.Println("No existing blockchain database found, create a new one first")
		runtime.Goexit()
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	Handle(err)

	var lastHash []byte
	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...) // Make a copy of the value
			return nil
		})
		return err
	})

	chain := &BlockChain{LastHash: lastHash, Database: db}
	return chain

}

func (chain *BlockChain) AddBlock(transactions []*Transaction) error {
	var lastHash []byte

	for _, tx := range transactions {
		if chain.VerifyTransaction(tx) != true {
			log.Panic("Invalid Transaction")
		}
	}

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		return item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})
	})
	Handle(err)

	newBlock := CreateBlock(transactions, lastHash)
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	return err
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{
		CurrentHash: chain.LastHash,
		Database:    chain.Database,
	}
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block
	var encondBlock []byte
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		return item.Value(func(val []byte) error {
			encondBlock = val
			return nil
		})
	})
	Handle(err)
	block = Deserialize(encondBlock)
	iter.CurrentHash = block.PrevHash
	return block
}

func DBExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (chain *BlockChain) FindUnspentTransaction(publicKeyHash []byte) []Transaction {
	var unspentTransactions []Transaction
	spentTransaction := make(map[string][]int)
	iterator := chain.Iterator()

	for {
		bloco := iterator.Next()

		for _, tx := range bloco.Transactions {
			id := hex.EncodeToString(tx.ID)

		Outputs:
			for outputID, output := range tx.Outputs {
				if spentTransaction[id] != nil {
					for _, spentOut := range spentTransaction[id] {
						if spentOut == outputID {
							continue Outputs
						}
					}
				}
				if output.IsLocked(publicKeyHash) {
					unspentTransactions = append(unspentTransactions, *tx)
				}
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.UsesKey(publicKeyHash) {
						inTransactionID := hex.EncodeToString(in.ID)
						spentTransaction[inTransactionID] = append(spentTransaction[inTransactionID], in.Out)
					}
				}
			}
		}

		if len(bloco.PrevHash) == 0 {
			break
		}
	}
	fmt.Printf("spent transactions %x\n", unspentTransactions)
	return unspentTransactions
}

func (chain *BlockChain) FindUTXO(publicKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransaction(publicKeyHash)

	for _, tx := range unspentTransactions {
		for _, output := range tx.Outputs {
			if output.IsLocked(publicKeyHash) {
				UTXOs = append(UTXOs, output)
			}
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(publicKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	fmt.Printf("PublicKey da transfericia de dinheiro: %x\n", publicKeyHash)
	unspentTransactions := chain.FindUnspentTransaction(publicKeyHash)
	saldo := 0
Work:
	for _, tx := range unspentTransactions {
		id := hex.EncodeToString(tx.ID)

		for outputID, output := range tx.Outputs {
			if output.IsLocked(publicKeyHash) && saldo < amount {
				saldo += output.Value
				unspentOuts[id] = append(unspentOuts[id], outputID)
				if saldo >= amount {
					break Work
				}
			}
		}
	}
	return saldo, unspentOuts
}

func (chain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iterator := chain.Iterator()

	for {
		bloco := iterator.Next()

		for _, tx := range bloco.Transactions {
			if hex.EncodeToString(tx.ID) == hex.EncodeToString(ID) {
				return *tx, nil
			}
		}
		if len(bloco.PrevHash) == 0 {
			break
		}
	}
	return Transaction{}, fmt.Errorf("Transaction not found")
}

func (chain *BlockChain) SignTransaction(t *Transaction, privateKey ecdsa.PrivateKey) {
	previousTransaction := make(map[string]Transaction)
	for _, input := range t.Inputs {
		tx, err := chain.FindTransaction(input.ID)
		Handle(err)
		previousTransaction[hex.EncodeToString(tx.ID)] = tx
	}
	t.Sign(privateKey, previousTransaction)
}

func (chain *BlockChain) VerifyTransaction(t *Transaction) bool {
	previousTransaction := make(map[string]Transaction)
	for _, input := range t.Inputs {
		tx, err := chain.FindTransaction(input.ID)
		Handle(err)
		previousTransaction[hex.EncodeToString(tx.ID)] = tx
	}
	return t.Verify(previousTransaction)
}
