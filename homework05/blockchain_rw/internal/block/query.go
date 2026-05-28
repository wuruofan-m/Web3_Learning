package block

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

// BlockInfo 包含区块的关键信息
type BlockInfo struct {
	Number     uint64
	Hash       string
	ParentHash string
	Time       uint64
	TxCount    int
	Coinbase   string
	GasLimit   uint64
	GasUsed    uint64
}

func (b *BlockInfo) String() string {
	return fmt.Sprintf(
		"Block #%d\n  Hash:        %s\n  ParentHash:  %s\n  Time:        %d\n  TxCount:     %d\n  Coinbase:    %s\n  GasLimit:    %d\n  GasUsed:     %d",
		b.Number, b.Hash, b.ParentHash, b.Time, b.TxCount, b.Coinbase, b.GasLimit, b.GasUsed,
	)
}

// GetBlockByNumber 按区块号查询区块详情，number 为 nil 时查询最新区块
func GetBlockByNumber(client *ethclient.Client, number *big.Int) (*BlockInfo, error) {
	blk, err := client.BlockByNumber(context.Background(), number)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	return &BlockInfo{
		Number:     blk.NumberU64(),
		Hash:       blk.Hash().Hex(),
		ParentHash: blk.ParentHash().Hex(),
		Time:       blk.Time(),
		TxCount:    len(blk.Transactions()),
		Coinbase:   blk.Coinbase().Hex(),
		GasLimit:   blk.GasLimit(),
		GasUsed:    blk.GasUsed(),
	}, nil
}

// GetLatestBlock 查询最新区块
func GetLatestBlock(client *ethclient.Client) (*BlockInfo, error) {
	return GetBlockByNumber(client, nil)
}
