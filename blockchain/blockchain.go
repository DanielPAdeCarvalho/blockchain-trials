package blockchain

import (
	"encoding/hex"
	"fmt"
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
		chain.LastHash,
		chain.Database,
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

func (chain *BlockChain) FindUnspentTransaction(address string) []Transaction {
	var unspentTransactions []Transaction
	spentTransaction := make(map[string][]int)
	iterator := chain.Iterator()

	for {
		bloco := iterator.Next()

		for _, tx := range bloco.Transaction {
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
				if output.CanUnlock(address) {
					unspentTransactions = append(unspentTransactions, *tx)
				}
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
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
	return unspentTransactions
}

func (chain *BlockChain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransaction(address)

	for _, tx := range unspentTransactions {
		for _, output := range tx.Outputs {
			if output.CanUnlock(address) {
				UTXOs = append(UTXOs, output)
			}
		}
	}

	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTransactions := chain.FindUnspentTransaction(address)
	saldo := 0

Work:
	for _, tx := range unspentTransactions {
		id := hex.EncodeToString(tx.ID)
		for outputID, output := range tx.Outputs {
			if output.CanUnlock(address) && saldo < amount {
				unspentOuts[id] = append(unspentOuts[id], outputID)
				saldo += output.Value
				if saldo >= amount {
					break Work
				}
			}
		}
	}
	return saldo, unspentOuts
}
