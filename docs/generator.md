# Generated bindings

The plugin reads ordinary protobuf `service`/`rpc` declarations and emits Go
client and server bindings beside the message code. RPC subjects use the proto
package, service and method names. Only Protobuf descriptors and the native
runtime API are used: there are no legacy nRPC options, wire compatibility
wrappers, or gRPC dependencies.

All four standard protobuf shapes are recognized. Unary uses the runtime's
one-request/one-reply operation. Server streaming, client streaming and bidi
use the same versioned conversation wire in `github.com/gopherex/natsrpc`.
Generated method signatures remain typed; the generator does not select,
require, or implement a payload codec. Both generated constructors require an
application codec. Use the same codec and queue group arrangement across
replicas; a JSON codec honors generated `MarshalJSON`/`UnmarshalJSON` methods.

`make test` verifies descriptor-driven output. `make integration` compiles
and runs the generated sample against Docker NATS with two independent,
explicitly injected codecs: Protobuf and standard JSON. Runtime reliability,
ACL, process failure and load tests belong to the runtime repository, not to
the generator.
