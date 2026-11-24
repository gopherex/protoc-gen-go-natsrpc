package generator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func fixture(t *testing.T, client, server bool) *protogen.Plugin {
	t.Helper()
	options := &descriptorpb.FileOptions{GoPackage: proto.String("example.com/rpc;rpc")}
	p, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"rpc.proto"},
		Parameter:      proto.String("paths=source_relative"), ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name: proto.String("rpc.proto"), Package: proto.String("rpc"), Syntax: proto.String("proto3"), Options: options,
			MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Req")}, {Name: proto.String("Resp")}},
			Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Greeter"), Method: []*descriptorpb.MethodDescriptorProto{{
				Name: proto.String("Hello"), InputType: proto.String(".rpc.Req"), OutputType: proto.String(".rpc.Resp"),
				ClientStreaming: proto.Bool(client), ServerStreaming: proto.Bool(server),
			}}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	plugin := fixture(t, false, false)
	if err := Generate(plugin); err != nil {
		t.Fatal(err)
	}
	result := plugin.Response()
	if result.GetError() != "" || len(result.GetFile()) != 1 {
		t.Fatalf("response: %v", result)
	}
	file := result.GetFile()[0]
	if file.GetName() != "rpc_natsrpc.pb.go" {
		t.Fatalf("filename %s", file.GetName())
	}
	for _, want := range []string{
		`Service: "Greeter"`, `Subject("Hello", 0`,
		"type GreeterNATSClient struct", "func NewGreeterNATSClient(",
		"type GreeterNATSHandler interface", "func RegisterGreeterNATS(",
	} {
		if !strings.Contains(file.GetContent(), want) {
			t.Errorf("missing %q\n%s", want, file.GetContent())
		}
	}
}

func TestNativeCardinality(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		client, server bool
		want           string
	}{
		{"unary", false, false, "natsrpc.Unary"},
		{"server", false, true, "natsrpc.Streamed"},
		{"client", true, false, "natsrpc.ClientStreamed"},
		{"bidi", true, true, "natsrpc.Bidirectional"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plug := fixture(t, tc.client, tc.server)
			if err := Generate(plug); err != nil {
				t.Fatal(err)
			}
			result := plug.Response()
			if result.GetError() != "" || len(result.GetFile()) != 1 || !strings.Contains(result.GetFile()[0].GetContent(), tc.want) {
				t.Fatalf("expected binding using %s: %v", tc.want, result)
			}
		})
	}
}
