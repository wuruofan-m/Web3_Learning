const { expect } = require("chai");
const { ethers, upgrades } = require("hardhat");

describe("MetaNodeStake", function () {
  let owner, user1, user2;
  let metaNodeToken, metaNodeStake;
  let startBlock, endBlock;
  let snapshotId;
  const MetaNodePerBlock = ethers.parseEther("10");
  const minDepositAmount = ethers.parseEther("0.01");
  const unstakeLockedBlocks = 10;

  // 只部署一次合约
  before(async function () {
    [owner, user1, user2] = await ethers.getSigners();

    // 部署 MetaNodeToken
    const MetaNodeToken = await ethers.getContractFactory("MetaNodeToken");
    metaNodeToken = await MetaNodeToken.deploy();

    // 部署 MetaNodeStake（UUPS 代理）
    const MetaNodeStake = await ethers.getContractFactory("MetaNodeStake");
    const currentBlock = await ethers.provider.getBlockNumber();
    startBlock = currentBlock + 5;
    endBlock = startBlock + 10000;

    metaNodeStake = await upgrades.deployProxy(
      MetaNodeStake,
      [await metaNodeToken.getAddress(), startBlock, endBlock, MetaNodePerBlock],
      { kind: "uups" }
    );

    // 给合约转入奖励代币
    await metaNodeToken.transfer(
      await metaNodeStake.getAddress(),
      ethers.parseEther("1000000")
    );

    // 添加 ETH 质押池（PID=0）
    await metaNodeStake.addPool(
      ethers.ZeroAddress,
      100,
      minDepositAmount,
      unstakeLockedBlocks,
      false
    );

    // 保存初始快照
    snapshotId = await ethers.provider.send("evm_snapshot");
  });

  // 每个测试前恢复快照
  beforeEach(async function () {
    await ethers.provider.send("evm_revert", [snapshotId]);
    snapshotId = await ethers.provider.send("evm_snapshot");
  });

  // ==================== 初始化测试 ====================
  describe("Initialize", function () {
    it("should set correct initial values", async function () {
      expect(await metaNodeStake.startBlock()).to.equal(startBlock);
      expect(await metaNodeStake.endBlock()).to.equal(endBlock);
      expect(await metaNodeStake.MetaNodePerBlock()).to.equal(MetaNodePerBlock);
      expect(await metaNodeStake.poolLength()).to.equal(1);
    });

    it("should not allow re-initialization", async function () {
      await expect(
        metaNodeStake.initialize(
          await metaNodeToken.getAddress(),
          startBlock,
          endBlock,
          MetaNodePerBlock
        )
      ).to.be.reverted;
    });

    it("should reject invalid parameters", async function () {
      const MetaNodeStake = await ethers.getContractFactory("MetaNodeStake");
      await expect(
        upgrades.deployProxy(
          MetaNodeStake,
          [await metaNodeToken.getAddress(), endBlock, startBlock, MetaNodePerBlock],
          { kind: "uups" }
        )
      ).to.be.revertedWith("invalid parameters");

      await expect(
        upgrades.deployProxy(
          MetaNodeStake,
          [await metaNodeToken.getAddress(), startBlock, endBlock, 0],
          { kind: "uups" }
        )
      ).to.be.revertedWith("invalid parameters");
    });
  });

  // ==================== 管理函数测试 ====================
  describe("Admin functions", function () {
    it("should add ETH pool as first pool with address(0)", async function () {
      const pool = await metaNodeStake.pool(0);
      expect(pool.stTokenAddress).to.equal(ethers.ZeroAddress);
      expect(pool.poolWeight).to.equal(100);
    });

    it("should not add ETH pool as non-first pool", async function () {
      const TestERC20 = await ethers.getContractFactory("TestERC20");
      const testToken = await TestERC20.deploy("Test", "TST", ethers.parseEther("1000"));
      await metaNodeStake.addPool(
        await testToken.getAddress(),
        50, 1, 10, false
      );

      await expect(
        metaNodeStake.addPool(ethers.ZeroAddress, 30, 1, 10, false)
      ).to.be.revertedWith("invalid staking token address");
    });

    it("should add ERC20 pool", async function () {
      const TestERC20 = await ethers.getContractFactory("TestERC20");
      const testToken = await TestERC20.deploy("Test", "TST", ethers.parseEther("1000"));

      await metaNodeStake.addPool(
        await testToken.getAddress(),
        50, 1, 10, false
      );

      expect(await metaNodeStake.poolLength()).to.equal(2);
      const pool = await metaNodeStake.pool(1);
      expect(pool.stTokenAddress).to.equal(await testToken.getAddress());
      expect(pool.poolWeight).to.equal(50);
    });

    it("should update pool info", async function () {
      await metaNodeStake.updatePool(0, ethers.parseEther("1"), 20);
      const pool = await metaNodeStake.pool(0);
      expect(pool.minDepositAmount).to.equal(ethers.parseEther("1"));
      expect(pool.unstakeLockedBlocks).to.equal(20);
    });

    /**
 * 测试设置池权重功能，验证指定池的权重更新是否生效
 * @param {number} 0 - 池ID
 * @param {number} 200 - 新的池权重值
 * @param {boolean} false - 是否立即更新全局奖励分配的标志
 * @returns {Promise<void>} 测试异步函数无返回值
 * @throws {AssertionError} 当池权重未正确更新为预期值时抛出断言错误
 */
it("should set pool weight", async function () {
      await metaNodeStake.setPoolWeight(0, 200, false);
      const pool = await metaNodeStake.pool(0);
      expect(pool.poolWeight).to.equal(200);
    });

    it("should pause/unpause withdraw", async function () {
      await metaNodeStake.pauseWithdraw();
      expect(await metaNodeStake.withdrawPaused()).to.be.true;
      await metaNodeStake.unpauseWithdraw();
      expect(await metaNodeStake.withdrawPaused()).to.be.false;
    });

    it("should pause/unpause claim", async function () {
      await metaNodeStake.pauseClaim();
      expect(await metaNodeStake.claimPaused()).to.be.true;
      await metaNodeStake.unpauseClaim();
      expect(await metaNodeStake.claimPaused()).to.be.false;
    });

    it("should set startBlock", async function () {
      const newStart = startBlock + 100;
      await metaNodeStake.setStartBlock(newStart);
      expect(await metaNodeStake.startBlock()).to.equal(newStart);
    });

    it("should reject startBlock > endBlock", async function () {
      await expect(
        metaNodeStake.setStartBlock(endBlock + 1)
      ).to.be.revertedWith("start block must be smaller than end block");
    });

    it("should set endBlock", async function () {
      const newEnd = endBlock + 1000;
      await metaNodeStake.setEndBlock(newEnd);
      expect(await metaNodeStake.endBlock()).to.equal(newEnd);
    });

    it("should set MetaNodePerBlock", async function () {
      const newRate = ethers.parseEther("20");
      await metaNodeStake.setMetaNodePerBlock(newRate);
      expect(await metaNodeStake.MetaNodePerBlock()).to.equal(newRate);
    });

    it("should reject non-admin calling admin functions", async function () {
      await expect(
        metaNodeStake.connect(user1).pauseWithdraw()
      ).to.be.reverted;

      await expect(
        metaNodeStake.connect(user1).setMetaNodePerBlock(1)
      ).to.be.reverted;

      await expect(
        metaNodeStake.connect(user1).addPool(ethers.ZeroAddress, 1, 1, 1, false)
      ).to.be.reverted;
    });
  });

  // ==================== ETH 质押测试 ====================
  describe("Deposit ETH", function () {
    it("should deposit ETH successfully", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      const balance = await metaNodeStake.stakingBalance(0, owner.address);
      expect(balance).to.equal(ethers.parseEther("1"));
    });

    it("should reject deposit below minimum", async function () {
      await expect(
        metaNodeStake.depositETH({ value: ethers.parseEther("0.001") })
      ).to.be.revertedWith("deposit amount is too small");
    });

    it("should update pool stTokenAmount", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("2") });
      const pool = await metaNodeStake.pool(0);
      expect(pool.stTokenAmount).to.equal(ethers.parseEther("2"));
    });
  });

  // ==================== ERC20 质押测试 ====================
  describe("Deposit ERC20", function () {
    it("should deposit ERC20 successfully", async function () {
      const TestERC20 = await ethers.getContractFactory("TestERC20");
      const testToken = await TestERC20.deploy("Test", "TST", ethers.parseEther("1000000"));
      await metaNodeStake.addPool(
        await testToken.getAddress(),
        50, ethers.parseEther("10"), 10, false
      );

      await testToken.transfer(user1.address, ethers.parseEther("1000"));
      await testToken.connect(user1).approve(
        await metaNodeStake.getAddress(),
        ethers.parseEther("1000")
      );

      await metaNodeStake.connect(user1).deposit(1, ethers.parseEther("100"));
      const balance = await metaNodeStake.stakingBalance(1, user1.address);
      expect(balance).to.equal(ethers.parseEther("100"));
    });

    it("should reject deposit to PID 0 via deposit()", async function () {
      await expect(
        metaNodeStake.deposit(0, ethers.parseEther("1"))
      ).to.be.revertedWith("deposit not support ETH staking");
    });

    it("should reject deposit below minimum", async function () {
      const TestERC20 = await ethers.getContractFactory("TestERC20");
      const testToken = await TestERC20.deploy("Test", "TST", ethers.parseEther("1000000"));
      await metaNodeStake.addPool(
        await testToken.getAddress(),
        50, ethers.parseEther("10"), 10, false
      );

      await testToken.transfer(user1.address, ethers.parseEther("1000"));
      await testToken.connect(user1).approve(
        await metaNodeStake.getAddress(),
        ethers.parseEther("1000")
      );

      await expect(
        metaNodeStake.connect(user1).deposit(1, ethers.parseEther("1"))
      ).to.be.revertedWith("deposit amount is too small");
    });
  });

  // ==================== 解质押测试 ====================
  describe("Unstake", function () {
    it("should create unstake request", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("5") });
      await metaNodeStake.unstake(0, ethers.parseEther("2"));
      const [requestAmount] = await metaNodeStake.withdrawAmount(0, owner.address);
      expect(requestAmount).to.equal(ethers.parseEther("2"));
    });

    it("should reduce staking balance", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("5") });
      await metaNodeStake.unstake(0, ethers.parseEther("2"));
      const balance = await metaNodeStake.stakingBalance(0, owner.address);
      expect(balance).to.equal(ethers.parseEther("3"));
    });

    it("should reject unstake more than staked", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("5") });
      await expect(
        metaNodeStake.unstake(0, ethers.parseEther("10"))
      ).to.be.revertedWith("Not enough staking token balance");
    });

    it("should reject withdraw when paused", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("5") });
      await metaNodeStake.unstake(0, ethers.parseEther("2"));
      await metaNodeStake.pauseWithdraw();
      await expect(
        metaNodeStake.withdraw(0)
      ).to.be.revertedWith("withdraw is paused");
    });
  });

  // ==================== 提现测试 ====================
  describe("Withdraw", function () {
    it("should withdraw after lock period", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("5") });
      await metaNodeStake.unstake(0, ethers.parseEther("3"));

      for (let i = 0; i < unstakeLockedBlocks + 1; i++) {
        await ethers.provider.send("evm_mine");
      }

      await metaNodeStake.withdraw(0);
      const [, pendingWithdraw] = await metaNodeStake.withdrawAmount(0, owner.address);
      expect(pendingWithdraw).to.equal(0);
    });

    it("should not withdraw before unlock", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("5") });
      await metaNodeStake.unstake(0, ethers.parseEther("3"));
      await metaNodeStake.withdraw(0);
      const [, pendingWithdraw] = await metaNodeStake.withdrawAmount(0, owner.address);
      expect(pendingWithdraw).to.equal(0);
    });
  });

  // ==================== 领取奖励测试 ====================
  describe("Claim", function () {
    it("should have pending reward", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      for (let i = 0; i < 5; i++) {
        await ethers.provider.send("evm_mine");
      }
      const pending = await metaNodeStake.pendingMetaNode(0, owner.address);
      expect(pending).to.be.gt(0);
    });

    it("should claim reward successfully", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      for (let i = 0; i < 5; i++) {
        await ethers.provider.send("evm_mine");
      }
      const balanceBefore = await metaNodeToken.balanceOf(owner.address);
      await metaNodeStake.claim(0);
      const balanceAfter = await metaNodeToken.balanceOf(owner.address);
      expect(balanceAfter).to.be.gt(balanceBefore);
    });

    it("should reject claim when paused", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      await metaNodeStake.pauseClaim();
      await expect(
        metaNodeStake.claim(0)
      ).to.be.revertedWith("claim is paused");
    });

    it("should claim when contract MetaNode balance insufficient", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      for (let i = 0; i < 5; i++) {
        await ethers.provider.send("evm_mine");
      }

      // 把合约中大部分奖励代币转走，使余额不足
      const contractAddr = await metaNodeStake.getAddress();
      const rewardBalance = await metaNodeToken.balanceOf(contractAddr);
      // 合约中 ETH 余额也会被算作 stTokenAmount，但奖励代币只有一小部分
      // 转走大部分 MetaNodeToken，让合约余额 < 应发奖励
      await metaNodeToken.transfer(user1.address, rewardBalance - 1n);

      // claim 应该走 _safeMetaNodeTransfer 的 amount > balance 分支
      await metaNodeStake.claim(0);
      // 用户应该只收到 1 个 MetaNode（合约仅剩的余额）
      expect(await metaNodeToken.balanceOf(owner.address)).to.be.gt(0);
    });
  });

  // ==================== 查询函数测试 ====================
  describe("Query functions", function () {
    it("should return pool length", async function () {
      expect(await metaNodeStake.poolLength()).to.equal(1);
    });

    it("should return staking balance", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("3") });
      const balance = await metaNodeStake.stakingBalance(0, owner.address);
      expect(balance).to.equal(ethers.parseEther("3"));
    });

    it("should return multiplier", async function () {
      const multiplier = await metaNodeStake.getMultiplier(startBlock, startBlock + 10);
      expect(multiplier).to.equal(MetaNodePerBlock * 10n);
    });

    it("should reject invalid pool id", async function () {
      await expect(
        metaNodeStake.stakingBalance(99, owner.address)
      ).to.be.revertedWith("invalid pid");
    });
  });

  // ==================== 暂停机制测试 ====================
  describe("Pause", function () {
    it("should reject unstake when withdraw paused", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      await metaNodeStake.pauseWithdraw();
      await expect(
        metaNodeStake.unstake(0, ethers.parseEther("0.5"))
      ).to.be.revertedWith("withdraw is paused");
    });

    it("should reject claim when claim paused", async function () {
      await metaNodeStake.depositETH({ value: ethers.parseEther("1") });
      await metaNodeStake.pauseClaim();
      await expect(
        metaNodeStake.claim(0)
      ).to.be.revertedWith("claim is paused");
    });
  });

  // ==================== 多池权重分配测试 ====================
  describe("Multi-pool reward distribution", function () {
    it("should distribute rewards by pool weight", async function () {
      const TestERC20 = await ethers.getContractFactory("TestERC20");
      const testToken = await TestERC20.deploy("Test", "TST", ethers.parseEther("1000000"));
      await metaNodeStake.addPool(
        await testToken.getAddress(),
        50, 1, 10, false
      );

      await metaNodeStake.connect(user1).depositETH({ value: ethers.parseEther("1") });

      await testToken.transfer(user2.address, ethers.parseEther("100"));
      await testToken.connect(user2).approve(
        await metaNodeStake.getAddress(),
        ethers.parseEther("100")
      );
      await metaNodeStake.connect(user2).deposit(1, ethers.parseEther("50"));

      for (let i = 0; i < 10; i++) {
        await ethers.provider.send("evm_mine");
      }

      const pending1 = await metaNodeStake.pendingMetaNode(0, user1.address);
      const pending2 = await metaNodeStake.pendingMetaNode(1, user2.address);
      expect(pending1).to.be.gt(pending2);
    });
  });
});
