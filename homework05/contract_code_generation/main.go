package main

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"

	"contract_code_generation/counter"
)

func main() {
	// ========================================
	// 1. 创建模拟区块链环境
	// ========================================
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("生成私钥失败: %v", err)
	}

	chainID := big.NewInt(1337)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("创建交易签名器失败: %v", err)
	}

	// 给账户分配初始 ETH
	alloc := make(core.GenesisAlloc)
	alloc[auth.From] = core.GenesisAccount{
		Balance: big.NewInt(1000000000000000000), // 1 ETH
	}
	blockchain := backends.NewSimulatedBackend(alloc, 8000000)
	defer blockchain.Close()

	fmt.Println("========================================")
	fmt.Println("  Counter 智能合约测试 (Simulated Backend)")
	fmt.Println("========================================")
	fmt.Printf("部署账户地址: %s\n\n", auth.From.Hex())

	// ========================================
	// 2. 部署合约 (初始计数 = 5)
	// ========================================
	fmt.Println("--- 部署合约 ---")
	address, tx, instance, err := counter.DeployCounter(auth, blockchain, big.NewInt(5))
	if err != nil {
		log.Fatalf("部署合约失败: %v", err)
	}
	blockchain.Commit()

	fmt.Printf("合约地址: %s\n", address.Hex())
	fmt.Printf("部署交易: %s\n", tx.Hash().Hex())

	// 验证初始计数
	count, err := instance.GetCount(nil)
	if err != nil {
		log.Fatalf("读取计数失败: %v", err)
	}
	fmt.Printf("初始计数: %s\n\n", count.String())

	// 查询合约所有者
	owner, err := instance.Owner(nil)
	if err != nil {
		log.Fatalf("读取所有者失败: %v", err)
	}
	fmt.Printf("合约所有者: %s\n\n", owner.Hex())

	// ========================================
	// 3. 测试 increment (递增)
	// ========================================
	fmt.Println("--- 测试 increment ---")
	for i := 0; i < 3; i++ {
		tx, err := instance.Increment(auth)
		if err != nil {
			log.Fatalf("increment 失败: %v", err)
		}
		blockchain.Commit()

		count, err = instance.GetCount(nil)
		if err != nil {
			log.Fatalf("读取计数失败: %v", err)
		}
		fmt.Printf("increment 后计数: %s (tx: %s)\n", count.String(), tx.Hash().Hex())
	}

	// ========================================
	// 4. 测试 decrement (递减)
	// ========================================
	fmt.Println("\n--- 测试 decrement ---")
	for i := 0; i < 2; i++ {
		tx, err := instance.Decrement(auth)
		if err != nil {
			log.Fatalf("decrement 失败: %v", err)
		}
		blockchain.Commit()

		count, err = instance.GetCount(nil)
		if err != nil {
			log.Fatalf("读取计数失败: %v", err)
		}
		fmt.Printf("decrement 后计数: %s (tx: %s)\n", count.String(), tx.Hash().Hex())
	}

	// ========================================
	// 5. 测试 decrement 下溢保护
	// ========================================
	fmt.Println("\n--- 测试 decrement 下溢保护 ---")
	// 先 reset 到 0
	_, err = instance.Reset(auth)
	if err != nil {
		log.Fatalf("reset 失败: %v", err)
	}
	blockchain.Commit()
	count, _ = instance.GetCount(nil)
	fmt.Printf("reset 后计数: %s\n", count.String())

	// 尝试在 0 时 decrement，应该失败
	_, err = instance.Decrement(auth)
	blockchain.Commit()
	if err != nil {
		fmt.Printf("decrement 在 0 时正确失败: %v\n", err)
	} else {
		fmt.Println("警告: decrement 在 0 时应该失败但成功了!")
	}

	// ========================================
	// 6. 测试 reset (重置)
	// ========================================
	fmt.Println("\n--- 测试 reset ---")
	// 先 increment 几次
	_, err = instance.Increment(auth)
	if err != nil {
		log.Fatalf("increment 失败: %v", err)
	}
	_, err = instance.Increment(auth)
	if err != nil {
		log.Fatalf("increment 失败: %v", err)
	}
	blockchain.Commit()
	count, _ = instance.GetCount(nil)
	fmt.Printf("increment x2 后计数: %s\n", count.String())

	// reset
	tx, err = instance.Reset(auth)
	if err != nil {
		log.Fatalf("reset 失败: %v", err)
	}
	blockchain.Commit()
	count, err = instance.GetCount(nil)
	if err != nil {
		log.Fatalf("读取计数失败: %v", err)
	}
	fmt.Printf("reset 后计数: %s (tx: %s)\n", count.String(), tx.Hash().Hex())

	// ========================================
	// 7. 测试 transferOwnership (转移所有权)
	// ========================================
	fmt.Println("\n--- 测试 transferOwnership ---")
	newPrivateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("生成新私钥失败: %v", err)
	}
	newOwnerAddress := crypto.PubkeyToAddress(newPrivateKey.PublicKey)

	tx, err = instance.TransferOwnership(auth, newOwnerAddress)
	if err != nil {
		log.Fatalf("transferOwnership 失败: %v", err)
	}
	blockchain.Commit()

	owner, err = instance.Owner(nil)
	if err != nil {
		log.Fatalf("读取所有者失败: %v", err)
	}
	fmt.Printf("旧所有者: %s\n", auth.From.Hex())
	fmt.Printf("新所有者: %s (tx: %s)\n", owner.Hex(), tx.Hash().Hex())

	// ========================================
	// 8. 测试权限控制：旧所有者不能再 reset
	// ========================================
	fmt.Println("\n--- 测试权限控制 (旧所有者调用 reset) ---")
	_, err = instance.Reset(auth)
	blockchain.Commit()
	if err != nil {
		fmt.Printf("旧所有者调用 reset 正确失败: %v\n", err)
	} else {
		fmt.Println("警告: 旧所有者调用 reset 应该失败但成功了!")
	}

	// ========================================
	// 测试结果汇总
	// ========================================
	fmt.Println("\n========================================")
	fmt.Println("  所有测试完成!")
	fmt.Println("========================================")
	finalCount, _ := instance.GetCount(nil)
	finalOwner, _ := instance.Owner(nil)
	fmt.Printf("最终计数: %s\n", finalCount.String())
	fmt.Printf("最终所有者: %s\n", finalOwner.Hex())
}
