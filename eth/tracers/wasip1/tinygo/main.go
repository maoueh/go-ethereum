package main

// Compile this file with 'tinygo build -o tracer.wasm -scheduler=none --no-debug -target=wasi main.go'

import "C"

import (
	"encoding/hex"
	"fmt"
	"unsafe"

	pbtracing "github.com/ethereum/go-ethereum/eth/tracers/wasip1/pb/eth/tracing/v1"
	"google.golang.org/protobuf/proto"
)

// main is required for TinyGo to compile to Wasm.
func main() {
}

//export onBlockchainInit
func onBlockchainInit(ptr, size uint32) {
	msg := unmarshalMessage(ptr, size, &pbtracing.OnBlockchainInitMessage{})

	fmt.Printf("Blockchain Init %d\n", msg.ChainConfig.ChainId.ToBig().Uint64())
}

//export onEnter
func onEnter(ptr, size uint32) {
	msg := unmarshalMessage(ptr, size, &pbtracing.OnEnterMessage{})

	fmt.Printf("Enter %s\n", hex.EncodeToString(msg.From))
}

//export onExit
func onExit(ptr, size uint32) {
	msg := unmarshalMessage(ptr, size, &pbtracing.OnExitMessage{})

	fmt.Printf("Exit %d\n", msg.Depth)
}

//export onOpcode
func onOpcode(ptr, size uint32) {
	msg := unmarshalMessage(ptr, size, &pbtracing.OnOpcodeMessage{})

	fmt.Printf("OnOpcode %d\n", msg.OpCode)
}

//export onFault
func onFault(ptr, size uint32) {
	msg := unmarshalMessage(ptr, size, &pbtracing.OnFaultMessage{})

	fmt.Printf("OnFault %d\n", msg.OpCode)
}

//export onGasChange
func onGasChange(ptr, size uint32) {
	msg := unmarshalMessage(ptr, size, &pbtracing.OnGasChangeMessage{})

	fmt.Printf("onGasChange %d\n", msg.New)
}

func unmarshalMessage[T VTUnmarshaller](ptr, size uint32, into T) T {
	data := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), size)
	if err := into.UnmarshalVT(data); err != nil {
		panic(err)
	}

	return into
}

type VTUnmarshaller interface {
	proto.Message
	UnmarshalVT([]byte) error
}
