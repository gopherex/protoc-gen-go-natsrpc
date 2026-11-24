# protoc-gen-go-natsrpc

The generator emits typed bindings and never supplies an application codec.
Do not write runtime, nRPC, or transport code in this repository. Regenerate
checked-in examples through EasyP rather than editing generated Go.

| Task | Tool |
| --- | --- |
| Protobuf frontend | `google.golang.org/protobuf/compiler/protogen` |
| Regenerate | `make gen` |
| Verify generated code | `make test`, `make lint`, `make integration` |
