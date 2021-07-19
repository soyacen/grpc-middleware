package grpcoteltrace

import (
	"google.golang.org/grpc/metadata"
)

// MetaDataCarrier adapts metadata.MD to satisfy the TextMapCarrier interface.
type MetaDataCarrier metadata.MD

// Get returns the value associated with the passed key.
func (mc MetaDataCarrier) Get(key string) string {
	vals := metadata.MD(mc).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set stores the key-value pair.
func (mc MetaDataCarrier) Set(key string, value string) {
	metadata.MD(mc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (mc MetaDataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range mc {
		keys = append(keys, k)
	}
	return keys
}
