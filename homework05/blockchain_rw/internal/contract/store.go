package contract

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SimpleStorageABI 是 SimpleStorage 合约的 ABI JSON
const SimpleStorageABI = `[
  {
    "inputs": [],
    "name": "get",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "_value",
        "type": "uint256"
      }
    ],
    "name": "set",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "newValue",
        "type": "uint256"
      }
    ],
    "name": "ValueChanged",
    "type": "event"
  }
]`

// SimpleStorageBin 是合约编译后的字节码
// 运行 `make compile && make abigen` 生成，或手动将编译后的 bytecode 填入此处
var SimpleStorageBin = ""

// SimpleStorage 是合约交互封装
type SimpleStorage struct {
	contract *bind.BoundContract
	address  common.Address
	client   *ethclient.Client
	parsedAbi abi.ABI
}

// NewSimpleStorage 创建已部署合约的交互实例
func NewSimpleStorage(address common.Address, client *ethclient.Client) (*SimpleStorage, error) {
	parsedAbi, err := abi.JSON(strings.NewReader(SimpleStorageABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	boundContract := bind.NewBoundContract(address, parsedAbi, client, client, client)

	return &SimpleStorage{
		contract:  boundContract,
		address:   address,
		client:    client,
		parsedAbi: parsedAbi,
	}, nil
}

// Address 返回合约地址
func (s *SimpleStorage) Address() common.Address {
	return s.address
}

// DeploySimpleStorage 部署新的 SimpleStorage 合约
func DeploySimpleStorage(auth *bind.TransactOpts, client *ethclient.Client) (common.Address, *types.Transaction, *SimpleStorage, error) {
	if SimpleStorageBin == "" {
		return common.Address{}, nil, nil, fmt.Errorf("contract bytecode is empty; please run 'make compile' first and update SimpleStorageBin in store.go")
	}

	parsedAbi, err := abi.JSON(strings.NewReader(SimpleStorageABI))
	if err != nil {
		return common.Address{}, nil, nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	address, tx, _, err := bind.DeployContract(auth, parsedAbi, common.FromHex(SimpleStorageBin), client)
	if err != nil {
		return common.Address{}, nil, nil, fmt.Errorf("failed to deploy contract: %w", err)
	}

	instance, err := NewSimpleStorage(address, client)
	if err != nil {
		return common.Address{}, nil, nil, fmt.Errorf("failed to create contract instance: %w", err)
	}

	return address, tx, instance, nil
}

// Get 调用合约的 get() 方法，读取存储值
func (s *SimpleStorage) Get(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := s.contract.Call(opts, &out, "get")
	if err != nil {
		return nil, fmt.Errorf("failed to call get(): %w", err)
	}

	if len(out) == 0 {
		return big.NewInt(0), nil
	}

	result, ok := out[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected return type from get()")
	}

	return result, nil
}

// Set 调用合约的 set(uint256) 方法，写入存储值
func (s *SimpleStorage) Set(opts *bind.TransactOpts, value *big.Int) (*types.Transaction, error) {
	tx, err := s.contract.Transact(opts, "set", value)
	if err != nil {
		return nil, fmt.Errorf("failed to call set(): %w", err)
	}

	return tx, nil
}

// WaitMined 等待交易确认
func (s *SimpleStorage) WaitMined(tx *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(context.Background(), s.client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}
	return receipt, nil
}
