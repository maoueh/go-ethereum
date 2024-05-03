package tracers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	pbtracing "github.com/ethereum/go-ethereum/eth/tracers/wasip1/pb/eth/tracing/v1"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"google.golang.org/protobuf/proto"
)

type traceConfig struct {
	WASMPath string `json:"wasm_path"`
}

func init() {
	LiveDirectory.Register("wasip1", NewWasiP1TracerFromRawConfig)
}

func NewWasiP1TracerFromRawConfig(config json.RawMessage) (*tracing.Hooks, error) {
	var cfg traceConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	tracer := &WASMP1Tracer{
		runtimeCtx:    runtimeCtx,
		runtimeCancel: runtimeCancel,
	}

	if err := tracer.configureWASM(&cfg); err != nil {
		return nil, fmt.Errorf("creating new WASM tracer: %w", err)
	}

	hooks := &tracing.Hooks{
		OnClose: func() {
			runtimeCancel()
			tracer.runtimeCloser.Close(context.Background())
		},
	}

	tracer.onBlockainInit = tracer.module.ExportedFunction("onBlockchainInit")
	tracer.onEnter = tracer.module.ExportedFunction("onEnter")
	tracer.onExit = tracer.module.ExportedFunction("onExit")
	tracer.onOpcode = tracer.module.ExportedFunction("onOpcode")
	tracer.onFault = tracer.module.ExportedFunction("onFault")
	tracer.onGasChange = tracer.module.ExportedFunction("onGasChange")

	if tracer.onBlockainInit != nil {
		hooks.OnBlockchainInit = tracer.OnBlockchainInit
	}

	if tracer.onEnter != nil {
		hooks.OnEnter = tracer.OnEnter
	}

	if tracer.onExit != nil {
		hooks.OnExit = tracer.OnExit
	}

	if tracer.onOpcode != nil {
		hooks.OnOpcode = tracer.OnOpcode
	}

	if tracer.onFault != nil {
		hooks.OnFault = tracer.OnFault
	}

	if tracer.onGasChange != nil {
		hooks.OnGasChange = tracer.OnGasChange
	}

	return hooks, nil
}

type WASMP1Tracer struct {
	runtime       wazero.Runtime
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc
	runtimeCloser api.Closer
	module        api.Module
	memory        Memory

	onBlockainInit api.Function
	onEnter        api.Function
	onExit         api.Function
	onOpcode       api.Function
	onFault        api.Function
	onGasChange    api.Function
}

func (t *WASMP1Tracer) configureWASM(config *traceConfig) error {
	wasmCode, err := os.ReadFile(config.WASMPath)
	if err != nil {
		return fmt.Errorf("reading WASM file: %w", err)
	}

	t.runtime = wazero.NewRuntime(t.runtimeCtx)
	t.runtimeCloser, err = wasi_snapshot_preview1.Instantiate(t.runtimeCtx, t.runtime)
	if err != nil {
		return fmt.Errorf("instantiating WASI: %w", err)
	}

	compiledModule, err := t.runtime.CompileModule(t.runtimeCtx, wasmCode)
	if err != nil {
		return fmt.Errorf("creating new module: %w", err)
	}

	moduleExportedFunctions := compiledModule.ExportedFunctions()
	if _, ok := moduleExportedFunctions["onBlockchainInit"]; !ok {
		return fmt.Errorf("missing required function 'onBlockchainInit'")
	}

	moduleConfig := wazero.NewModuleConfig().WithStdout(os.Stdout).WithStderr(os.Stderr).WithStartFunctions("initialize")

	t.module, err = t.runtime.InstantiateModule(t.runtimeCtx, compiledModule, moduleConfig)
	if err != nil {
		return fmt.Errorf("instantiating module: %w", err)
	}

	// Memory allocation is runtime specific
	t.memory, err = NewTinyGoMemory(t.module.ExportedFunction("malloc"), t.module.ExportedFunction("free"), t.module.Memory())
	if err != nil {
		return fmt.Errorf("creating new memory: %w", err)
	}

	return nil
}

func (t *WASMP1Tracer) OnBlockchainInit(chainConfig *params.ChainConfig) {
	t.invokeWasm(t.onBlockainInit, &pbtracing.OnBlockchainInitMessage{
		ChainConfig: new(pbtracing.ChainConfig).FromEth(chainConfig),
	})
}

func (t *WASMP1Tracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.invokeWasm(t.onEnter, &pbtracing.OnEnterMessage{
		Depth:  int64(depth),
		OpCode: uint64(typ),
		From:   from.Bytes(),
		To:     to.Bytes(),
		Input:  input,
		Gas:    gas,
		Value:  pbtracing.NewBigIntFromBig(value),
	})
}
func (t *WASMP1Tracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	t.invokeWasm(t.onExit, &pbtracing.OnExitMessage{
		Depth:    int64(depth),
		Output:   output,
		GasUsed:  gasUsed,
		Error:    errorToProto(err),
		Reverted: reverted,
	})
}

func (t *WASMP1Tracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	t.invokeWasm(t.onOpcode, &pbtracing.OnOpcodeMessage{
		Pc:     pc,
		OpCode: uint64(op),
		Gas:    gas,
		Cost:   cost,
		RData:  rData,
		Depth:  int32(depth),
		Error:  errorToProto(err),
	})
}

func (t *WASMP1Tracer) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	t.invokeWasm(t.onFault, &pbtracing.OnFaultMessage{
		Pc:     pc,
		OpCode: uint64(op),
		Gas:    gas,
		Cost:   cost,
		Depth:  int32(depth),
		Error:  errorToProto(err),
	})
}

func (t *WASMP1Tracer) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	t.invokeWasm(t.onGasChange, &pbtracing.OnGasChangeMessage{
		Old:    old,
		New:    new,
		Reason: uint64(reason),
	})
}

func (t *WASMP1Tracer) invokeWasm(function api.Function, msg proto.Message) {
	if err := t.invokeWasmExecutor(function, msg); err != nil {
		panic(err)
	}
}

func (t *WASMP1Tracer) invokeWasmExecutor(function api.Function, msg proto.Message) error {
	dataBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshalling message %T: %w", msg, err)
	}

	dataPtr, free, err := t.memory.Write(t.runtimeCtx, dataBytes)
	if err != nil {
		return fmt.Errorf("writing data to memory: %w", err)
	}
	defer free()

	_, err = function.Call(t.runtimeCtx, uint64(dataPtr), uint64(len(dataBytes)))
	if err != nil {
		return fmt.Errorf("calling function: %w", err)
	}

	return nil
}

func errorToProto(err error) *string {
	if err == nil {
		return nil
	}

	s := err.Error()
	return &s
}

// Memory is an interface for memory management since it differs depending on target
// WASM runtime.
type Memory interface {
	Write(ctx context.Context, data []byte) (ptr uint32, free func() error, err error)
}

var _ Memory = (*TinyGoMemory)(nil)

type TinyGoMemory struct {
	allocator   api.Function
	deallocator api.Function
	memory      api.Memory
}

func NewTinyGoMemory(allocator, deallocator api.Function, memory api.Memory) (*TinyGoMemory, error) {
	if allocator == nil {
		return nil, errors.New("something is wrong, allocator function 'malloc' is nil, TinyGo is supposed to provide it")
	}

	if deallocator == nil {
		return nil, errors.New("something is wrong, allocator function 'free' is nil, TinyGo is supposed to provide it")
	}

	return &TinyGoMemory{
		allocator:   allocator,
		deallocator: deallocator,
		memory:      memory,
	}, nil
}

// Write implements Memory.
func (t *TinyGoMemory) Write(ctx context.Context, data []byte) (ptr uint32, free func() error, err error) {
	results, err := t.allocator.Call(ctx, uint64(len(data)))
	if err != nil {
		return 0, nil, fmt.Errorf("allocation of %d bytes failed within runtime allocator: %w", len(data), err)
	}

	dataPtr := results[0]
	free = func() error {
		_, err := t.deallocator.Call(ctx, dataPtr)
		return err
	}

	if !t.memory.Write(uint32(dataPtr), data) {
		free()
		return 0, nil, fmt.Errorf("memory write at offset %d with size %d (end offset %d) out of range of memory size %d",
			dataPtr, len(data), dataPtr+uint64(len(data)), t.memory.Size(),
		)
	}

	return uint32(dataPtr), free, nil
}
