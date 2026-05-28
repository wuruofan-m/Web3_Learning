package client

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

const SepoliaChainID = 11155111

// Connect 建立以太坊客户端连接，并验证是否为 Sepolia 网络
func Connect(rpcURL string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if chainID.Cmp(big.NewInt(SepoliaChainID)) != 0 {
		return nil, fmt.Errorf("connected to wrong network: expected chain ID %d, got %d (make sure you are using Sepolia testnet)", SepoliaChainID, chainID)
	}

	return client, nil
}

// MustConnect 连接失败则 panic
func MustConnect(rpcURL string) *ethclient.Client {
	client, err := Connect(rpcURL)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Ethereum client: %v", err))
	}
	return client
}
