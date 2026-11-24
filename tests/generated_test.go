//go:build integration

package tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/proto"

	"github.com/gopherex/natsrpc"
	"github.com/gopherex/natsrpc/codec/jsoncodec"

	"github.com/gopherex/protoc-gen-go-natsrpc/example"
)

func TestGeneratedBindings(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	broker, err := testcontainers.Run(ctx, "nats:2.15.3-alpine", testcontainers.WithExposedPorts("4222/tcp"),
		testcontainers.WithWaitStrategy(wait.ForAll(
			wait.ForLog("Server is ready"), wait.ForListeningPort("4222/tcp").SkipInternalCheck())))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(broker) })
	url, err := broker.PortEndpoint(ctx, "4222/tcp", "nats")
	if err != nil {
		t.Fatal(err)
	}
	binary, err := natsrpc.NewFuncCodec(marshalProto, unmarshalProto)
	if err != nil {
		t.Fatal(err)
	}
	checkGenerated(t, ctx, url, binary)
	checkGenerated(t, ctx, url, jsoncodec.Codec{})
}

func marshalProto(value any) ([]byte, error) {
	switch msg := value.(type) {
	case proto.Message:
		return proto.Marshal(msg)
	default:
		return nil, fmt.Errorf("not a Protobuf message: %T", value)
	}
}

func unmarshalProto(data []byte, into any) error {
	switch msg := into.(type) {
	case proto.Message:
		return proto.Unmarshal(data, msg)
	default:
		return fmt.Errorf("not a Protobuf message: %T", into)
	}
}

type cardinality struct{}

func (cardinality) Echo(_ context.Context, req *example.Chunk) (*example.Chunk, error) {
	if req.GetText() == "refuse" {
		return nil, &natsrpc.Error{Code: natsrpc.CodeNotFound, Message: "example refusal", Detail: &example.Chunk{Text: "binary detail"}}
	}
	return &example.Chunk{Text: req.GetText()}, nil
}

func (cardinality) List(_ context.Context, req *example.Chunk, send func(*example.Chunk) error) error {
	for _, text := range strings.Split(req.GetText(), ",") {
		if err := send(&example.Chunk{Text: text}); err != nil {
			return err
		}
	}
	return nil
}

func (cardinality) Collect(_ context.Context, recv func() (*example.Chunk, error)) (*example.Chunk, error) {
	var chunks []string
	for {
		message, err := recv()
		if errors.Is(err, io.EOF) {
			return &example.Chunk{Text: strings.Join(chunks, ",")}, nil
		}
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, message.GetText())
	}
}

func (cardinality) Chat(_ context.Context, recv func() (*example.Chunk, error), send func(*example.Chunk) error) error {
	if err := send(&example.Chunk{Text: "ready"}); err != nil {
		return err
	}
	for {
		message, err := recv()
		if errors.Is(err, io.EOF) {
			return send(&example.Chunk{Text: "done"})
		}
		if err != nil {
			return err
		}
		if err := send(&example.Chunk{Text: "echo:" + message.GetText()}); err != nil {
			return err
		}
	}
}

func checkGenerated(t *testing.T, parent context.Context, url string, codec natsrpc.Codec) {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()
	conn, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reg, err := example.RegisterCardinalityNATS(ctx, conn, cardinality{}, codec, natsrpc.ServerConfig{QueueGroup: "example.replicas"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer stopCancel()
		if err := reg.Drain(stopCtx); err != nil {
			t.Error(err)
		}
	}()
	client, err := example.NewCardinalityNATSClient(conn, codec, natsrpc.ClientConfig{})
	if err != nil {
		t.Fatal(err)
	}
	unary, err := client.Echo(ctx, &example.Chunk{Text: "echo"})
	if err != nil || unary.GetText() != "echo" {
		t.Fatalf("unary %v %v", unary, err)
	}
	checkGeneratedRefusal(t, ctx, client)
	var listed []string
	err = client.List(ctx, &example.Chunk{Text: "a,b"}, func(msg *example.Chunk) error { listed = append(listed, msg.GetText()); return nil })
	if err != nil || strings.Join(listed, ",") != "a,b" {
		t.Fatalf("server-stream %v %v", listed, err)
	}
	collecting, err := client.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer collecting.Close()
	for _, text := range []string{"x", "y"} {
		if err := collecting.Send(ctx, &example.Chunk{Text: text}); err != nil {
			t.Fatal(err)
		}
	}
	collected, err := collecting.CloseAndRecv(ctx)
	if err != nil || collected.GetText() != "x,y" {
		t.Fatalf("client-stream %v %v", collected, err)
	}
	checkGeneratedChat(t, ctx, client)
}

func checkGeneratedChat(t *testing.T, ctx context.Context, client *example.CardinalityNATSClient) {
	t.Helper()
	chat, err := client.Chat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer chat.Close()
	if initial, err := chat.Recv(ctx); err != nil || initial.GetText() != "ready" {
		t.Fatalf("bidi ready %v %v", initial, err)
	}
	if err := chat.Send(ctx, &example.Chunk{Text: "request"}); err != nil {
		t.Fatal(err)
	}
	if echo, err := chat.Recv(ctx); err != nil || echo.GetText() != "echo:request" {
		t.Fatalf("bidi reply %v %v", echo, err)
	}
	if err := chat.CloseSend(ctx); err != nil {
		t.Fatal(err)
	}
	if done, err := chat.Recv(ctx); err != nil || done.GetText() != "done" {
		t.Fatalf("bidi done %v %v", done, err)
	}
	if _, err := chat.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func checkGeneratedRefusal(t *testing.T, ctx context.Context, client *example.CardinalityNATSClient) {
	t.Helper()
	for _, remoteCancel := range []bool{false, true} {
		opts := []natsrpc.CallOption{natsrpc.WithErrorDetail(func() *example.Chunk { return new(example.Chunk) })}
		if remoteCancel {
			opts = append(opts, natsrpc.WithRemoteCancel())
		}
		response, err := client.Echo(ctx, &example.Chunk{Text: "refuse"}, opts...)
		var refusal *natsrpc.DetailedError[*example.Chunk]
		if response != nil || !errors.As(err, &refusal) || refusal.Failure.Code != natsrpc.CodeNotFound || refusal.Detail.GetText() != "binary detail" {
			t.Fatalf("typed refusal, remote cancellation %v: %v %v", remoteCancel, response, err)
		}
	}
}
