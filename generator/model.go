// Package generator emits typed Go bindings for the native NATS RPC wire.
package generator

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/gopherex/natsrpc"
)

type servicePlan struct {
	svc     *protogen.Service
	route   natsrpc.Route
	methods []methodPlan
}

func (plan servicePlan) bindingName() string { return plan.svc.GoName + "NATS" }

type methodPlan struct {
	method           *protogen.Method
	subject          string
	stream, incoming bool
}

func planService(file *protogen.File, svc *protogen.Service) (servicePlan, error) {
	route := natsrpc.Route{Prefix: string(file.Desc.Package()), Service: string(svc.Desc.Name())}
	if _, err := route.Subscription(); err != nil {
		return servicePlan{}, fmt.Errorf("%s: %w", svc.Desc.FullName(), err)
	}

	result := servicePlan{svc: svc, route: route, methods: make([]methodPlan, 0, len(svc.Methods))}

	names := make(map[string]bool, len(svc.Methods))
	for _, method := range svc.Methods {
		if names[method.GoName] {
			return servicePlan{}, fmt.Errorf("%s: %w: duplicate generated method",
				method.Desc.FullName(), natsrpc.ErrConfiguration)
		}

		names[method.GoName] = true

		subject := string(method.Desc.Name())
		if _, err := route.Subject(subject, 0); err != nil {
			return servicePlan{}, fmt.Errorf("%s: %w", method.Desc.FullName(), err)
		}

		result.methods = append(result.methods, methodPlan{
			method: method, subject: subject,
			stream: method.Desc.IsStreamingServer(), incoming: method.Desc.IsStreamingClient(),
		})
	}

	return result, nil
}
