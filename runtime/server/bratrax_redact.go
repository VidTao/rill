package server

import (
	"strings"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Bratrax runs the runtime multi-tenant with auth disabled, so every handler
// response reaches the tenant's browser. Instance and connector config carry
// real secrets (the client's BYOK Anthropic key, the shared ClickHouse
// password), which the runtime reads from the instance directly and the
// frontend never uses. These helpers blank them in responses only; they
// always copy, because the inputs are the live instance's own values.

var secretKeyParts = []string{"password", "secret", "token", "api_key", "apikey", "private_key", "credentials"}

func isSecretKey(key string) bool {
	key = strings.ToLower(key)
	for _, p := range secretKeyParts {
		if strings.Contains(key, p) {
			return true
		}
	}
	return false
}

func redactStruct(s *structpb.Struct) *structpb.Struct {
	if s == nil {
		return nil
	}
	s = proto.Clone(s).(*structpb.Struct)
	for k := range s.Fields {
		if isSecretKey(k) {
			s.Fields[k] = structpb.NewStringValue("")
		}
	}
	return s
}

func redactStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	res := make(map[string]string, len(m))
	for k, v := range m {
		if isSecretKey(k) {
			v = ""
		}
		res[k] = v
	}
	return res
}

func redactConnectors(cs []*runtimev1.Connector) []*runtimev1.Connector {
	if cs == nil {
		return nil
	}
	res := make([]*runtimev1.Connector, len(cs))
	for i, c := range cs {
		c = proto.Clone(c).(*runtimev1.Connector)
		c.Config = redactStruct(c.Config)
		res[i] = c
	}
	return res
}
