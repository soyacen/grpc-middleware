package grpclogger

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type FieldBuilder struct {
	fields map[string]interface{}
}

func NewFieldBuilder() *FieldBuilder {
	return &FieldBuilder{fields: make(map[string]interface{})}
}

func (f *FieldBuilder) Server() *FieldBuilder {
	f.fields["system"] = "grpc.server"
	return f
}

func (f *FieldBuilder) Client() *FieldBuilder {
	f.fields["system"] = "grpc.client"
	return f
}

func (f *FieldBuilder) StartTime(startTime time.Time) *FieldBuilder {
	f.fields["grpc.start_time"] = startTime.Format(time.RFC3339)
	return f
}

func (f *FieldBuilder) Deadline(ctx context.Context) *FieldBuilder {
	if d, ok := ctx.Deadline(); ok {
		f.fields["grpc.request.deadline"] = d.Format(time.RFC3339)
	}
	return f
}

func (f *FieldBuilder) Latency(duration time.Duration) *FieldBuilder {
	f.fields["grpc.latency"] = duration.String()
	return f
}

func (f *FieldBuilder) Method(method string) *FieldBuilder {
	f.fields["grpc.method"] = method
	return f
}

func (f *FieldBuilder) PeerAddr(ctx context.Context) *FieldBuilder {
	if peer, ok := peer.FromContext(ctx); ok {
		f.fields["grpc.peer.address"] = peer.Addr.String()
	}
	return f
}

func (f *FieldBuilder) MetaData(md metadata.MD) *FieldBuilder {
	for key, val := range md {
		f.fields["grpc.md."+key] = val
	}
	return f
}

func (f *FieldBuilder) Status(err error) *FieldBuilder {
	f.fields["grpc.response.status"] = status.Code(err)
	return f
}

func (f *FieldBuilder) Error(err error) *FieldBuilder {
	if err == nil {
		return f
	}
	f.fields["error"] = err.Error()
	return f
}

func (f *FieldBuilder) Build() map[string]interface{} {
	return f.fields
}
