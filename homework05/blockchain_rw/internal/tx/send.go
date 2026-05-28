package tx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TxInfo 包含交易的关键信息
type TxInfo struct {
	Hash      string
	From      string
	To        string
	Value     string // Wei
	GasPrice  string
	GasLimit  uint64
	Nonce     uint64
	BlockHash string
	BlockNum  uint64
}

func (t *TxInfo) String() string {
	return fmt.Sprintf(
		"Transaction\n  Hash:      %s\n  From:      %s\n  To:        %s\n  Value:     %s Wei\n  GasPrice:  %s\n  GasLimit:  %d\n  Nonce:     %d\n  BlockHash: %s\n  BlockNum:  %d",
		t.Hash, t.From, t.To, t.Value, t.GasPrice, t.GasLimit, t.Nonce, t.BlockHash, t.BlockNum,
	)
}

// GetTxByHash 按哈希查询交易详情
func GetTxByHash(client *ethclient.Client, hash common.Hash) (*TxInfo, error) {
	ctx := context.Background()

	transaction, isPending, err := client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if isPending {
		return nil, fmt.Errorf("transaction %s is still pending", hash.Hex())
	}

	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt (may be pending): %w", err)
	}

	// 使用 TransactionSender 获取发送者地址
	fromAddr, err := client.TransactionSender(ctx, transaction, receipt.BlockHash, receipt.TransactionIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction sender: %w", err)
	}

	info := &TxInfo{
		Hash:      transaction.Hash().Hex(),
		From:      fromAddr.Hex(),
		Value:     transaction.Value().String(),
		GasPrice:  transaction.GasPrice().String(),
		GasLimit:  transaction.Gas(),
		Nonce:     transaction.Nonce(),
		BlockHash: receipt.BlockHash.Hex(),
		BlockNum:  receipt.BlockNumber.Uint64(),
	}

	if to := transaction.To(); to != nil {
		info.To = to.Hex()
	}

	return info, nil
}

// ParseEtherAmount 将 ETH 金额字符串转换为 Wei (*big.Int)
func ParseEtherAmount(amountStr string) (*big.Int, error) {
	amountFloat, ok := new(big.Float).SetString(amountStr)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", amountStr)
	}

	weiPerEth := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	amountInWei := new(big.Float).Mul(amountFloat, weiPerEth)
	result, accuracy := amountInWei.Int(nil)
	if accuracy == big.Above {
		return nil, fmt.Errorf("amount precision loss: %s", amountStr)
	}

	return result, nil
}

// NewTransactOpts 从私钥创建交易选项
func NewTransactOpts(client *ethclient.Client, privateKey *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	ctx := context.Background()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(300000) // 默认 gas limit，可被 EstimateGas 覆盖

	return auth, nil
}

// SendETH 向指定地址发送 ETH 转账交易
func SendETH(client *ethclient.Client, privateKey *ecdsa.PrivateKey, toAddress common.Address, amount *big.Int) (*types.Receipt, error) {
	ctx := context.Background()

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// 估算 gas
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:  fromAddress,
		To:    &toAddress,
		Value: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}
	// 增加 10% 缓冲
	gasLimit = gasLimit * 110 / 100

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	unsignedTx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, nil)

	signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	fmt.Printf("Transaction sent: %s\n", signedTx.Hash().Hex())
	fmt.Println("Waiting for confirmation...")

	receipt, err := bind.WaitMined(ctx, client, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction receipt: %w", err)
	}

	if receipt.Status == types.ReceiptStatusSuccessful {
		fmt.Printf("Transaction confirmed in block %d\n", receipt.BlockNumber.Uint64())
	} else {
		return receipt, fmt.Errorf("transaction failed (status: %d)", receipt.Status)
	}

	return receipt, nil
}

// GetBalance 查询指定地址的 ETH 余额
func GetBalance(client *ethclient.Client, address common.Address) (*big.Int, error) {
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

// WeiToEther 将 Wei 转换为 ETH 字符串
func WeiToEther(wei *big.Int) string {
	weiFloat := new(big.Float).SetInt(wei)
	ethValue := new(big.Float).Quo(weiFloat, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)))
	return fmt.Sprintf("%s Wei (%s ETH)", wei.String(), ethValue.Text('f', 18))
}
