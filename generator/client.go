package generator

import (
	"strconv"

	"google.golang.org/protobuf/compiler/protogen"
)

func emitClient(out *protogen.GeneratedFile, plan servicePlan, method methodPlan) {
	if method.incoming {
		emitConversationClient(out, plan, method)

		return
	}

	signature := clientSignature(out) + ", req " + requestType(out, method)

	name := method.method.GoName

	reply := out.QualifiedGoIdent(method.method.Output.GoIdent)

	callback := ", receive func(*" + reply + ") error"
	if method.stream {
		openClient(out, plan, method, name, signature+callback, "error", "err")
		out.P("return ", runtimeName(out, "Stream"), "(ctx, c.client, subject, req,",
			" func() *", reply, " { return new(", reply, ") }, receive, opts...)")
		out.P("}")
	} else {
		openClient(out, plan, method, name, signature, "(*"+reply+", error)", "nil, err")
		out.P("response := new(", reply, ")")
		out.P("if err := c.client.Call(ctx, subject, req, response, opts...); err != nil { return nil, err }")
		out.P("return response, nil")
		out.P("}")
	}
}

func clientSignature(out *protogen.GeneratedFile) string { return "ctx " + ctxType(out) }

func emitSubject(out *protogen.GeneratedFile, plan servicePlan, method methodPlan, fail string) {
	out.P("subject, err := route", plan.svc.GoName, "().Subject(", strconv.Quote(method.subject), ", 0)")
	out.P("if err != nil { return ", fail, " }")
}

func openClient(
	out *protogen.GeneratedFile, plan servicePlan, method methodPlan, name, signature, result, fail string,
) {
	out.P("func (c *", plan.svc.GoName, "Client) ", name, "(", signature,
		", opts ...", runtimeName(out, "CallOption"), ") ", result, " {")
	emitSubject(out, plan, method, fail)
}

func emitConversationClient(out *protogen.GeneratedFile, plan servicePlan, method methodPlan) {
	name := plan.svc.GoName + method.method.GoName + "Stream"

	open := "OpenClientStream"
	if method.stream {
		open = "OpenBidi"
	}

	out.P("type ", name, " struct { session *", runtimeName(out, "StreamSession"), " }")
	openClient(out, plan, method, method.method.GoName, clientSignature(out),
		"(*"+name+", error)", "nil, err")
	out.P("session, err := ", runtimeName(out, open), "(ctx, c.client, subject, opts...)")
	out.P("if err != nil { return nil, err }")
	out.P("return &", name, "{session: session}, nil")
	out.P("}")
	emitConversationMethods(out, method, name)
}

func emitConversationMethods(out *protogen.GeneratedFile, method methodPlan, name string) {
	ctx := ctxType(out)
	response := out.QualifiedGoIdent(method.method.Output.GoIdent)
	out.P("func (s *", name, ") Send(ctx ", ctx, ", req ", requestType(out, method), ") error {")
	out.P("return s.session.Send(ctx, req)")
	out.P("}")
	out.P("func (s *", name, ") Close() { s.session.Close() }")

	recv := "CloseAndRecv"
	if method.stream {
		recv = "Recv"

		out.P("func (s *", name, ") CloseSend(ctx ", ctx, ") error { return s.session.CloseSend(ctx) }")
	}

	out.P("func (s *", name, ") ", recv, "(ctx ", ctx, ") (*", response, ", error) {")
	out.P("response := new(", response, ")")
	out.P("if err := s.session.", recv, "(ctx, response); err != nil { return nil, err }")
	out.P("return response, nil")
	out.P("}")
}
