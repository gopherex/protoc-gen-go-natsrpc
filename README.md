# protoc-gen-go-natsrpc

A Go Protobuf compiler plugin that emits typed bindings for
[`github.com/gopherex/natsrpc`](https://github.com/gopherex/natsrpc). It generates
no codecs and has no legacy nRPC or gRPC transport dependency.

## Generate

Go 1.26+, `protoc-gen-go`, and either `protoc` or EasyP v1 are required.
Build the plugin with `make build`; `make gen` regenerates the example with
EasyP. With `protoc`:

```sh
PATH="$PWD/bin:$PATH" protoc -I . --go_out=paths=source_relative:. \
  --go-natsrpc_out=paths=source_relative:. example/cardinality.proto
```

## Bind and call

Supply a connected `conn`, your handler and a codec. The generator does not
choose JSON, binary Protobuf or any other format implicitly:

```go
codec := jsoncodec.Codec{}
registration, err := example.RegisterCardinality(ctx, conn, handler, codec,
    natsrpc.ServerConfig{QueueGroup: "cardinality-replicas"})
// Check err, then Drain the registration during shutdown.
client, err := example.NewCardinalityClient(conn, codec, natsrpc.ClientConfig{})
// Check err before calling client.Echo(ctx, &example.Chunk{Text: "hello"}).
```

`jsoncodec.Codec{}` is from `github.com/gopherex/natsrpc/codec/jsoncodec`. For
binary Protobuf, supply an adapter to `proto.Marshal` and `proto.Unmarshal`
using `natsrpc.NewFuncCodec`; see the working fixture in `tests/generated_test.go`.
The checked-in `example/cardinality.proto` demonstrates every RPC shape.

`make test`, `make lint` and `make integration` check code generation and all
four real-broker operations with explicitly supplied JSON and Protobuf codecs.
See [generator behavior](docs/generator.md) and the runtime repository for
wire, middleware, telemetry and operational guarantees.
