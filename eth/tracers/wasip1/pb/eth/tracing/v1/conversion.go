package pbtracing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func (c *ChainConfig) FromEth(config *params.ChainConfig) *ChainConfig {
	c.ChainId = NewBigIntFromBig(config.ChainID)
	c.HomesteadBlock = NewBigIntFromBig(config.HomesteadBlock)
	c.DaoForkBlock = NewBigIntFromBig(config.DAOForkBlock)
	c.DaoForkSupport = config.DAOForkSupport
	c.Eip150Block = NewBigIntFromBig(config.EIP150Block)
	c.Eip155Block = NewBigIntFromBig(config.EIP155Block)
	c.Eip158Block = NewBigIntFromBig(config.EIP158Block)
	c.ByzantiumBlock = NewBigIntFromBig(config.ByzantiumBlock)
	c.ConstantinopleBlock = NewBigIntFromBig(config.ConstantinopleBlock)
	c.PetersburgBlock = NewBigIntFromBig(config.PetersburgBlock)
	c.IstanbulBlock = NewBigIntFromBig(config.IstanbulBlock)
	c.MuirGlacierBlock = NewBigIntFromBig(config.MuirGlacierBlock)
	c.BerlinBlock = NewBigIntFromBig(config.BerlinBlock)
	c.LondonBlock = NewBigIntFromBig(config.LondonBlock)
	c.ArrowGlacierBlock = NewBigIntFromBig(config.ArrowGlacierBlock)
	c.GrayGlacierBlock = NewBigIntFromBig(config.GrayGlacierBlock)
	c.MergeNetsplitBlock = NewBigIntFromBig(config.MergeNetsplitBlock)
	c.ShanghaiTime = config.ShanghaiTime
	c.CancunTime = config.CancunTime
	c.PragueTime = config.PragueTime
	c.VerkleTime = config.VerkleTime
	c.TerminalTotalDifficulty = NewBigIntFromBig(config.TerminalTotalDifficulty)
	c.TerminalTotalDifficultyPassed = config.TerminalTotalDifficultyPassed

	return c
}

func NewBigIntFromBig(big *big.Int) *BigInt {
	if big == nil {
		return nil
	}

	return new(BigInt).FromBig(big)
}

func (i *BigInt) ToBig() *big.Int {
	if i == nil {
		return nil
	}

	return new(big.Int).SetBytes(i.Value)
}

func (i *BigInt) FromBig(big *big.Int) *BigInt {
	if big == nil {
		return i
	}

	i.Value = big.Bytes()
	return i
}

func (i *BigInt) FromUint256(big *uint256.Int) *BigInt {
	if big == nil {
		return nil
	}

	i.Value = big.Bytes()
	return i
}
