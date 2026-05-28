# blockchain_rw - Sepolia 区块链交互项目规格书

## 1. 项目概述

基于 Go 语言与 Solidity，在 Sepolia 测试网络上实现基础的区块链读写交互，包括：

- **查询区块**：按区块号获取区块信息
- **查询交易**：按交易哈希获取交易详情
- **查询余额**：查询指定地址的 ETH 余额
- **发送交易**：向指定地址转账 ETH
- **合约交互**：部署并调用简单 Solidity 合约（读写链上数据）

## 2. 技术栈

| 组件       | 技术选型                           |
|------------|------------------------------------|
| 后端语言   | Go 1.21+                           |
| 智能合约   | Solidity ^0.8.20                   |
| 区块链交互 | go-ethereum (geth) v1.13+          |
| 合约编译   | solc / abigen                      |
| 网络       | Sepolia 测试网                     |
| RPC        | Infura / Alchemy Sepolia Endpoint  |

## 3. 项目结构

```
blockchain_rw/
├── spec.md                  # 本规格书
├── main.go                  # 程序入口
├── go.mod
├── go.sum
├── internal/
│   ├── client/
│   │   └── eth_client.go    # 以太坊客户端封装
│   ├── block/
│   │   └── query.go         # 区块查询逻辑
│   ├── tx/
│   │   └── send.go          # 交易发送逻辑
│   └── contract/
│       ├── store.go         # 合约交互封装
│       └── store.abi        # 合约 ABI
├── contracts/
│   └── SimpleStorage.sol    # 简单存储合约
├── .env.example             # 环境变量示例
└── Makefile
```

## 4. 功能规格

### 4.1 以太坊客户端连接 (`internal/client/`)

- 通过 RPC URL 建立 `*ethclient.Client` 连接
- 验证连接：调用 `ChainID` 确认网络为 Sepolia (chainId = 11155111)
- 支持从环境变量读取 RPC URL

```go
// Connect 建立 ETH 客户端连接
func Connect(rpcURL string) (*ethclient.Client, error)

// MustConnect 连接失败则 panic
func MustConnect(rpcURL string) *ethclient.Client
```

### 4.2 区块查询 (`internal/block/`)

| 函数                              | 说明                     |
|-----------------------------------|--------------------------|
| `GetBlockByNumber(client, number)` | 按区块号查询区块详情     |
| `GetLatestBlock(client)`          | 查询最新区块             |

返回信息包括：
- 区块号 (Number)
- 时间戳 (Time)
- 父哈希 (ParentHash)
- 交易数量 (Tx count)
- 矿工地址 (Coinbase)
- Gas 限额 / 已用 (GasLimit / GasUsed)

### 4.3 交易发送 (`internal/tx/`)

| 函数                            | 说明                   |
|---------------------------------|------------------------|
| `SendETH(client, auth, to, amount)` | 发送 ETH 转账交易  |
| `GetTxByHash(client, hash)`    | 按哈希查询交易详情     |

发送交易流程：
1. 从私钥构造 `TransactOpts`（设置 Nonce、GasLimit、GasPrice）
2. 构建 `types.Transaction`
3. 签名并广播交易
4. 等待交易确认（`WaitMined`）
5. 返回交易回执

> ⚠️ 私钥仅从环境变量读取，禁止硬编码

### 4.4 合约交互 (`internal/contract/`)

#### 4.4.1 Solidity 合约 (`contracts/SimpleStorage.sol`)

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract SimpleStorage {
    uint256 private storedValue;

    event ValueChanged(uint256 newValue);

    function set(uint256 _value) external {
        storedValue = _value;
        emit ValueChanged(_value);
    }

    function get() external view returns (uint256) {
        return storedValue;
    }
}
```

#### 4.4.2 合约操作

| 操作   | 说明                                |
|--------|-------------------------------------|
| 部署   | 通过 `DeploySimpleStorage` 部署合约 |
| 写入   | 调用 `set(value)` 写入链上数据      |
| 读取   | 调用 `get()` 读取链上数据           |

#### 4.4.3 合约编译与绑定生成

```bash
# 编译合约生成 ABI 和 BIN
solc --abi --bin contracts/SimpleStorage.sol -o build/

# 生成 Go 绑定代码
abigen --abi=build/SimpleStorage.abi --bin=build/SimpleStorage.bin --pkg=contract --out=internal/contract/store.go
```

## 5. 环境变量 (`.env`)

```env
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/YOUR_PROJECT_ID
PRIVATE_KEY=your_private_key_here
```

- `SEPOLIA_RPC_URL`：Sepolia RPC 端点（Infura / Alchemy）
- `PRIVATE_KEY`：用于签名交易的私钥（仅测试用，带 ETH 的钱包）

## 6. 程序入口 (`cmd/main.go`)

程序以命令行子命令方式运行：

```bash
# 查询最新区块
go run main.go block latest

# 查询指定区块
go run main.go block 5000000

# 查询交易
go run main.go tx 0xabc123...

# 查询余额
go run main.go balance 0xYourAddress...

# 发送 ETH
go run main.go send 0xRecipientAddress 0.01

# 部署合约
go run main.go deploy

# 合约写入
go run main.go set 0xContractAddress 42

# 合约读取
go run main.go get 0xContractAddress
```

## 7. 错误处理

- RPC 连接失败：返回友好错误提示
- 交易发送失败：打印错误并退出（exit code 1）
- 合约调用失败：区分 revert 和网络错误
- 所有 `error` 必须被检查和处理，不允许 `_ = ...` 忽略

## 8. 依赖

```go
require (
    github.com/ethereum/go-ethereum v1.13.14
    github.com/joho/godotenv v1.5.1
)
```

## 9. 测试要求

- 使用 `github.com/stretchr/testify` 断言库
- 区块查询、余额查询等只读操作编写单元测试（可连接真实 Sepolia 或 mock）
- 交易发送和合约交互可手动测试，无需自动化测试

## 10. Makefile 命令

```makefile
build:        # 编译项目
run:          # 运行默认命令（查询最新区块）
compile:      # 编译 Solidity 合约
abigen:       # 生成 Go 绑定
clean:        # 清理构建产物
```

## 11. 安全注意事项

1. **私钥安全**：私钥仅从环境变量加载，`.env` 必须在 `.gitignore` 中
2. **Gas 估算**：发送交易前应估算 Gas，避免交易失败
3. **Nonce 管理**：使用 `PendingNonceAt` 获取待处理 Nonce
4. **测试网限定**：代码中硬编码检查 ChainID 必须为 Sepolia，防止误连主网
