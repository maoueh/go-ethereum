package wasip1_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestWasip1(t *testing.T) {
	context := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(uint64(1)),
		Time:        1,
		Difficulty:  big.NewInt(2),
		GasLimit:    uint64(1000000),
		BaseFee:     big.NewInt(8),
	}

	wasmPath, err := filepath.Abs("../../../wasip1/tinygo/tracer.wasm")
	require.NoError(t, err)

	hooks, err := tracers.NewWasiP1TracerFromRawConfig(json.RawMessage(fmt.Sprintf(`{
		"wasm_path": "%s"
	}`, wasmPath)))
	require.NoError(t, err)

	genesis, blockchain := newBlockchain(t, types.GenesisAlloc{}, context, hooks)

	block := types.NewBlock(&types.Header{
		ParentHash:       genesis.ToBlock().Hash(),
		Number:           context.BlockNumber,
		Difficulty:       context.Difficulty,
		Coinbase:         context.Coinbase,
		Time:             context.Time,
		GasLimit:         context.GasLimit,
		BaseFee:          context.BaseFee,
		ParentBeaconRoot: ptr(common.Hash{}),
	}, nil, nil, trie.NewStackTrie(nil))

	blockchain.SetBlockValidatorAndProcessorForTesting(
		ignoreValidateStateValidator{core.NewBlockValidator(genesis.Config, blockchain, blockchain.Engine())},
		core.NewStateProcessor(genesis.Config, blockchain, blockchain.Engine()),
	)

	n, err := blockchain.InsertChain(types.Blocks{block})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	hooks.OnClose()
}

func newBlockchain(t *testing.T, alloc types.GenesisAlloc, context vm.BlockContext, tracer *tracing.Hooks) (*core.Genesis, *core.BlockChain) {
	t.Helper()

	genesis := &core.Genesis{
		Difficulty: new(big.Int).Sub(context.Difficulty, big.NewInt(1)),
		Timestamp:  context.Time - 1,
		Number:     new(big.Int).Sub(context.BlockNumber, big.NewInt(1)).Uint64(),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Coinbase:   context.Coinbase,
		Config:     params.AllEthashProtocolChanges,
		Alloc:      alloc,
	}

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, false)))
	defer log.SetDefault(log.NewLogger(log.DiscardHandler()))

	blockchain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), core.DefaultCacheConfigWithScheme(rawdb.HashScheme), genesis, nil, ethash.NewFullFaker(), vm.Config{
		Tracer: tracer,
	}, nil, nil)
	require.NoError(t, err)

	return genesis, blockchain
}

type ignoreValidateStateValidator struct {
	core.Validator
}

func (v ignoreValidateStateValidator) ValidateBody(block *types.Block) error {
	return v.Validator.ValidateBody(block)
}

func (v ignoreValidateStateValidator) ValidateState(block *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {
	return nil
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		t.Fatal(err)
	}

	return !stat.IsDir()
}

func ptr[T any](v T) *T {
	return &v
}
