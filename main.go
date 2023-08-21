package main

import (
	"flag"
	"fmt"
	"golang-blockchain/blockchain"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct {
}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println(" print - Prints the blocks in the chain")
	fmt.Println(" getbalance -address ADDRESS - get the balance for")
	fmt.Println(" createblockchain -address ADDRESS creates a blockchain")
	fmt.Println(" printchain - Prints the blocks in the blockchain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT - Send amount")
}

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	iterator := chain.Iterator()

	for {
		bloco := iterator.Next()

		fmt.Printf("Previous hash: %s\n", bloco.PrevHash)
		fmt.Printf("Hash: %s\n", bloco.Hash)
		pow := blockchain.NewProof(bloco)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
	}
}

func (cli *CommandLine) run() {
	cli.validateArgs()
	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "Block data")
	switch os.Args[1] {
	case "add":
		err := addBlockCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		blockchain.Handle(err)
	default:
		cli.printUsage()
		runtime.Goexit()
	}
	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime.Goexit()
		}
		cli.addBlock(*addBlockData)
	}
	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func main() {
	chain := blockchain.InitBlockChain()
	defer chain.Database.Close()
	cli := CommandLine{chain}
	cli.run()
}
