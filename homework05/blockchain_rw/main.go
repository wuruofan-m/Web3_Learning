package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	"blockchain_rw/internal/block"
	"blockchain_rw/internal/client"
	"blockchain_rw/internal/contract"
	"blockchain_rw/internal/tx"
)

func main() {
	// 加载 .env
	_ = godotenv.Load()

	rpcURL := os.Getenv("SEPOLIA_RPC_URL")
	if rpcURL == "" {
		fmt.Println("Error: SEPOLIA_RPC_URL environment variable is not set")
		fmt.Println("Please set it in .env file or as environment variable")
		os.Exit(1)
	}

	ethClient := client.MustConnect(rpcURL)
	fmt.Println("Connected to Sepolia testnet")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "block":
		handleBlock(ethClient)
	case "tx":
		handleTx(ethClient)
	case "balance":
		handleBalance(ethClient)
	case "send":
		handleSend(ethClient)
	case "deploy":
		handleDeploy(ethClient)
	case "set":
		handleSet(ethClient)
	case "get":
		handleGet(ethClient)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: blockchain_rw <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  block latest       Query latest block")
	fmt.Println("  block <number>     Query block by number")
	fmt.Println("  tx <hash>          Query transaction by hash")
	fmt.Println("  balance <address>  Query ETH balance of address")
	fmt.Println("  send <to> <amount> Send ETH (amount in ETH, e.g. 0.01)")
	fmt.Println("  deploy             Deploy SimpleStorage contract")
	fmt.Println("  set <addr> <value> Set value in SimpleStorage contract")
	fmt.Println("  get <addr>         Get value from SimpleStorage contract")
}

func getPrivateKey() *ecdsa.PrivateKey {
	hexKey := os.Getenv("PRIVATE_KEY")
	if hexKey == "" {
		fmt.Println("Error: PRIVATE_KEY environment variable is not set")
		fmt.Println("Please set it in .env file or as environment variable")
		os.Exit(1)
	}

	privateKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		fmt.Printf("Error: invalid private key: %v\n", err)
		os.Exit(1)
	}

	return privateKey
}

func handleBlock(ethClient *ethclient.Client) {
	var blockInfo *block.BlockInfo
	var err error

	if len(os.Args) < 3 {
		fmt.Println("Usage: blockchain_rw block <number|latest>")
		os.Exit(1)
	}

	arg := os.Args[2]
	if arg == "latest" {
		blockInfo, err = block.GetLatestBlock(ethClient)
	} else {
		number := new(big.Int)
		number, ok := number.SetString(arg, 10)
		if !ok {
			fmt.Printf("Error: invalid block number: %s\n", arg)
			os.Exit(1)
		}
		blockInfo, err = block.GetBlockByNumber(ethClient, number)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(blockInfo.String())
}

func handleTx(ethClient *ethclient.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: blockchain_rw tx <txHash>")
		os.Exit(1)
	}

	txHash := common.HexToHash(os.Args[2])
	txInfo, err := tx.GetTxByHash(ethClient, txHash)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(txInfo.String())
}

func handleBalance(ethClient *ethclient.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: blockchain_rw balance <address>")
		os.Exit(1)
	}

	address := common.HexToAddress(os.Args[2])
	balance, err := tx.GetBalance(ethClient, address)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Balance of %s:\n  %s\n", address.Hex(), tx.WeiToEther(balance))
}

func handleSend(ethClient *ethclient.Client) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: blockchain_rw send <toAddress> <amountInETH>")
		os.Exit(1)
	}

	toAddress := common.HexToAddress(os.Args[2])
	amount, err := tx.ParseEtherAmount(os.Args[3])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	privateKey := getPrivateKey()

	receipt, err := tx.SendETH(ethClient, privateKey, toAddress, amount)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Transaction hash: %s\n", receipt.TxHash.Hex())
	fmt.Printf("Gas used: %d\n", receipt.GasUsed)
}

func handleDeploy(ethClient *ethclient.Client) {
	privateKey := getPrivateKey()

	auth, err := tx.NewTransactOpts(ethClient, privateKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	address, txHash, _, err := contract.DeploySimpleStorage(auth, ethClient)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Contract deployed!\n")
	fmt.Printf("  Contract Address: %s\n", address.Hex())
	fmt.Printf("  Transaction Hash: %s\n", txHash.Hash().Hex())
	fmt.Println("  Waiting for confirmation...")

	receipt, err := bindWaitMined(ethClient, txHash)
	if err != nil {
		fmt.Printf("Error waiting for deployment: %v\n", err)
		os.Exit(1)
	}

	if receipt.Status == 0 {
		fmt.Println("Contract deployment failed!")
		os.Exit(1)
	}

	fmt.Printf("  Confirmed in block: %d\n", receipt.BlockNumber.Uint64())
	fmt.Println()
	fmt.Printf("Use the following commands to interact:\n")
	fmt.Printf("  blockchain_rw set %s <value>\n", address.Hex())
	fmt.Printf("  blockchain_rw get %s\n", address.Hex())
}

func handleSet(ethClient *ethclient.Client) {
	if len(os.Args) < 4 {
		fmt.Println("Usage: blockchain_rw set <contractAddress> <value>")
		os.Exit(1)
	}

	contractAddr := common.HexToAddress(os.Args[2])
	value, ok := new(big.Int).SetString(os.Args[3], 10)
	if !ok {
		fmt.Printf("Error: invalid value: %s\n", os.Args[3])
		os.Exit(1)
	}

	privateKey := getPrivateKey()

	instance, err := contract.NewSimpleStorage(contractAddr, ethClient)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	auth, err := tx.NewTransactOpts(ethClient, privateKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	sentTx, err := instance.Set(auth, value)
	if err != nil {
		fmt.Printf("Error calling set(): %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set transaction sent: %s\n", sentTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	receipt, err := instance.WaitMined(sentTx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if receipt.Status == 0 {
		fmt.Println("Transaction failed!")
		os.Exit(1)
	}

	fmt.Printf("Value set successfully! Gas used: %d\n", receipt.GasUsed)
}

func handleGet(ethClient *ethclient.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: blockchain_rw get <contractAddress>")
		os.Exit(1)
	}

	contractAddr := common.HexToAddress(os.Args[2])

	instance, err := contract.NewSimpleStorage(contractAddr, ethClient)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	value, err := instance.Get(nil)
	if err != nil {
		fmt.Printf("Error calling get(): %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Stored value in contract %s: %s\n", contractAddr.Hex(), value.String())
}

func bindWaitMined(client *ethclient.Client, tx *types.Transaction) (*types.Receipt, error) {
	return bind.WaitMined(context.Background(), client, tx)
}
