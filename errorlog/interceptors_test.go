package errorlog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockLogHandler struct {
	records []map[string]interface{}
	attrs   []slog.Attr
	groups  []string
}

func (m *mockLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (m *mockLogHandler) Handle(ctx context.Context, r slog.Record) error {
	record := make(map[string]interface{})
	record["level"] = r.Level.String()
	record["message"] = r.Message
	r.Attrs(func(a slog.Attr) bool {
		record[a.Key] = a.Value.Any()
		return true
	})
	m.records = append(m.records, record)
	return nil
}

func (m *mockLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &mockLogHandler{
		records: m.records,
		attrs:   append(m.attrs, attrs...),
		groups:  m.groups,
	}
}

func (m *mockLogHandler) WithGroup(name string) slog.Handler {
	return &mockLogHandler{
		records: m.records,
		attrs:   m.attrs,
		groups:  append(m.groups, name),
	}
}

func setupMockLogger() (*mockLogHandler, func()) {
	handler := &mockLogHandler{records: make([]map[string]interface{}, 0)}
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	return handler, func() { slog.SetDefault(oldLogger) }
}

type mockUnaryHandler struct {
	resp      interface{}
	err       error
	callCount int
}

func (m *mockUnaryHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	m.callCount++
	return m.resp, m.err
}

type mockStreamHandler struct {
	err       error
	callCount int
}

func (m *mockStreamHandler) handle(srv interface{}, stream grpc.ServerStream) error {
	m.callCount++
	return m.err
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

type mockInvoker struct {
	err       error
	callCount int
	lastCtx   context.Context
}

func (m *mockInvoker) invoke(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	m.callCount++
	m.lastCtx = ctx
	return m.err
}

type mockStreamer struct {
	clientStream grpc.ClientStream
	err          error
	callCount    int
}

func (m *mockStreamer) stream(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	m.callCount++
	return m.clientStream, m.err
}

func TestUnaryServerInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		handlerErr     error
		wantErr        bool
		wantLogRecords int
	}{
		{
			name:           "成功请求不记录日志",
			handlerErr:     nil,
			wantErr:        false,
			wantLogRecords: 0,
		},
		{
			name:           "错误请求记录日志",
			handlerErr:     errors.New("handler error"),
			wantErr:        true,
			wantLogRecords: 1,
		},
		{
			name:           "gRPC状态错误记录日志",
			handlerErr:     status.Error(codes.Internal, "internal error"),
			wantErr:        true,
			wantLogRecords: 1,
		},
		{
			name:           "nil错误不记录日志",
			handlerErr:     nil,
			wantErr:        false,
			wantLogRecords: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockHandler := &mockUnaryHandler{err: tt.handlerErr}
			interceptor := UnaryServerInterceptor()
			resp, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, mockHandler.handle)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mockHandler.callCount)
			assert.Len(t, handler.records, tt.wantLogRecords)

			if tt.wantLogRecords > 0 {
				record := handler.records[0]
				assert.Equal(t, "ERROR", record["level"])
				assert.Equal(t, "gRPC call error", record["message"])
				assert.Equal(t, "unary", record["rpc_type"])
				assert.Equal(t, "server", record["system"])
				assert.Equal(t, "/test/Method", record["method"])
				assert.NotEmpty(t, record["error"])
				assert.NotEmpty(t, record["code"])
			}
		})
	}
}

func TestUnaryServerInterceptor_WithPrintRequest(t *testing.T) {
	tests := []struct {
		name          string
		printRequest  bool
		printResponse bool
		wantRequest   bool
		wantResponse  bool
	}{
		{
			name:          "打印请求和响应",
			printRequest:  true,
			printResponse: true,
			wantRequest:   true,
			wantResponse:  true,
		},
		{
			name:          "只打印请求",
			printRequest:  true,
			printResponse: false,
			wantRequest:   true,
			wantResponse:  false,
		},
		{
			name:          "只打印响应",
			printRequest:  false,
			printResponse: true,
			wantRequest:   false,
			wantResponse:  true,
		},
		{
			name:          "都不打印",
			printRequest:  false,
			printResponse: false,
			wantRequest:   false,
			wantResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockHandler := &mockUnaryHandler{resp: "response", err: errors.New("error")}
			interceptor := UnaryServerInterceptor(
				WithPrintRequest(tt.printRequest),
				WithPrintResponse(tt.printResponse),
			)
			_, _ = interceptor(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, mockHandler.handle)

			require.Len(t, handler.records, 1)
			record := handler.records[0]

			_, hasRequest := record["request"]
			_, hasResponse := record["response"]

			assert.Equal(t, tt.wantRequest, hasRequest)
			assert.Equal(t, tt.wantResponse, hasResponse)
		})
	}
}

func TestUnaryServerInterceptor_NilRequestResponse(t *testing.T) {
	tests := []struct {
		name          string
		printRequest  bool
		printResponse bool
		req           interface{}
		resp          interface{}
		wantRequest   bool
		wantResponse  bool
	}{
		{
			name:          "nil请求不打印即使开启",
			printRequest:  true,
			printResponse: true,
			req:           nil,
			resp:          nil,
			wantRequest:   false,
			wantResponse:  false,
		},
		{
			name:          "非nil请求和响应打印",
			printRequest:  true,
			printResponse: true,
			req:           "request",
			resp:          "response",
			wantRequest:   true,
			wantResponse:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockHandler := &mockUnaryHandler{resp: tt.resp, err: errors.New("error")}
			interceptor := UnaryServerInterceptor(
				WithPrintRequest(tt.printRequest),
				WithPrintResponse(tt.printResponse),
			)
			_, _ = interceptor(context.Background(), tt.req, &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, mockHandler.handle)

			require.Len(t, handler.records, 1)
			record := handler.records[0]

			_, hasRequest := record["request"]
			_, hasResponse := record["response"]

			assert.Equal(t, tt.wantRequest, hasRequest)
			assert.Equal(t, tt.wantResponse, hasResponse)
		})
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		handlerErr     error
		wantErr        bool
		wantLogRecords int
	}{
		{
			name:           "成功请求不记录日志",
			handlerErr:     nil,
			wantErr:        false,
			wantLogRecords: 0,
		},
		{
			name:           "错误请求记录日志",
			handlerErr:     errors.New("stream error"),
			wantErr:        true,
			wantLogRecords: 1,
		},
		{
			name:           "gRPC状态错误记录日志",
			handlerErr:     status.Error(codes.Internal, "internal stream error"),
			wantErr:        true,
			wantLogRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockHandler := &mockStreamHandler{err: tt.handlerErr}
			interceptor := StreamServerInterceptor()
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test/StreamMethod"}, mockHandler.handle)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mockHandler.callCount)
			assert.Len(t, handler.records, tt.wantLogRecords)

			if tt.wantLogRecords > 0 {
				record := handler.records[0]
				assert.Equal(t, "ERROR", record["level"])
				assert.Equal(t, "gRPC call error", record["message"])
				assert.Equal(t, "stream", record["rpc_type"])
				assert.Equal(t, "server", record["system"])
				assert.Equal(t, "/test/StreamMethod", record["method"])
			}
		})
	}
}

func TestStreamServerInterceptor_StreamContext(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		handlerErr error
	}{
		{
			name:       "流上下文传递",
			ctx:        context.WithValue(context.Background(), "key", "value"),
			handlerErr: errors.New("error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockHandler := &mockStreamHandler{err: tt.handlerErr}
			interceptor := StreamServerInterceptor()
			stream := &mockServerStream{ctx: tt.ctx}
			_ = interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test/Method"}, mockHandler.handle)

			require.Len(t, handler.records, 1)
			assert.Equal(t, "ERROR", handler.records[0]["level"])
		})
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		invokerErr     error
		wantErr        bool
		wantLogRecords int
	}{
		{
			name:           "成功调用不记录日志",
			invokerErr:     nil,
			wantErr:        false,
			wantLogRecords: 0,
		},
		{
			name:           "错误调用记录日志",
			invokerErr:     errors.New("invoker error"),
			wantErr:        true,
			wantLogRecords: 1,
		},
		{
			name:           "gRPC状态错误记录日志",
			invokerErr:     status.Error(codes.Unavailable, "service unavailable"),
			wantErr:        true,
			wantLogRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockInvoker := &mockInvoker{err: tt.invokerErr}
			interceptor := UnaryClientInterceptor()
			err := interceptor(context.Background(), "/test/Method", "request", "reply", nil, mockInvoker.invoke)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mockInvoker.callCount)
			assert.Len(t, handler.records, tt.wantLogRecords)

			if tt.wantLogRecords > 0 {
				record := handler.records[0]
				assert.Equal(t, "ERROR", record["level"])
				assert.Equal(t, "gRPC call error", record["message"])
				assert.Equal(t, "unary", record["rpc_type"])
				assert.Equal(t, "client", record["system"])
				assert.Equal(t, "/test/Method", record["method"])
			}
		})
	}
}

func TestUnaryClientInterceptor_WithOptions(t *testing.T) {
	tests := []struct {
		name          string
		printRequest  bool
		printResponse bool
		wantRequest   bool
		wantResponse  bool
	}{
		{
			name:          "打印请求和响应",
			printRequest:  true,
			printResponse: true,
			wantRequest:   true,
			wantResponse:  true,
		},
		{
			name:          "不打印请求和响应",
			printRequest:  false,
			printResponse: false,
			wantRequest:   false,
			wantResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockInvoker := &mockInvoker{err: errors.New("error")}
			interceptor := UnaryClientInterceptor(
				WithPrintRequest(tt.printRequest),
				WithPrintResponse(tt.printResponse),
			)
			_ = interceptor(context.Background(), "/test/Method", "request", "reply", nil, mockInvoker.invoke)

			require.Len(t, handler.records, 1)
			record := handler.records[0]

			_, hasRequest := record["request"]
			_, hasResponse := record["response"]

			assert.Equal(t, tt.wantRequest, hasRequest)
			assert.Equal(t, tt.wantResponse, hasResponse)
		})
	}
}

func TestStreamClientInterceptor(t *testing.T) {
	tests := []struct {
		name           string
		streamerErr    error
		wantErr        bool
		wantLogRecords int
		wantStream     bool
	}{
		{
			name:           "成功调用不记录日志",
			streamerErr:    nil,
			wantErr:        false,
			wantLogRecords: 0,
			wantStream:     true,
		},
		{
			name:           "错误调用记录日志",
			streamerErr:    errors.New("streamer error"),
			wantErr:        true,
			wantLogRecords: 1,
			wantStream:     false,
		},
		{
			name:           "gRPC状态错误记录日志",
			streamerErr:    status.Error(codes.Internal, "internal stream error"),
			wantErr:        true,
			wantLogRecords: 1,
			wantStream:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockStreamer := &mockStreamer{err: tt.streamerErr}
			interceptor := StreamClientInterceptor()
			stream, err := interceptor(context.Background(), nil, nil, "/test/Method", mockStreamer.stream)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, stream)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mockStreamer.callCount)
			assert.Len(t, handler.records, tt.wantLogRecords)

			if tt.wantLogRecords > 0 {
				record := handler.records[0]
				assert.Equal(t, "ERROR", record["level"])
				assert.Equal(t, "gRPC call error", record["message"])
				assert.Equal(t, "stream", record["rpc_type"])
				assert.Equal(t, "client", record["system"])
				assert.Equal(t, "/test/Method", record["method"])
			}
		})
	}
}

func TestStreamClientInterceptor_WithOptions(t *testing.T) {
	tests := []struct {
		name          string
		printRequest  bool
		printResponse bool
	}{
		{
			name:          "流式客户端选项不影响日志",
			printRequest:  true,
			printResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			mockStreamer := &mockStreamer{err: errors.New("error")}
			interceptor := StreamClientInterceptor(
				WithPrintRequest(tt.printRequest),
				WithPrintResponse(tt.printResponse),
			)
			_, _ = interceptor(context.Background(), nil, nil, "/test/Method", mockStreamer.stream)

			require.Len(t, handler.records, 1)
			record := handler.records[0]

			_, hasRequest := record["request"]
			_, hasResponse := record["response"]

			assert.False(t, hasRequest, "stream client should not have request")
			assert.False(t, hasResponse, "stream client should not have response")
		})
	}
}

func TestLogError_NonStatusError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantError  string
	}{
		{
			name:      "普通错误使用Unknown代码",
			err:       errors.New("plain error"),
			wantCode:  "Unknown",
			wantError: "plain error",
		},
		{
			name:      "状态错误使用对应代码",
			err:       status.Error(codes.Internal, "internal error"),
			wantCode:  "Internal",
			wantError: "rpc error: code = Internal desc = internal error",
		},
		{
			name:      "未找到错误",
			err:       status.Error(codes.NotFound, "not found"),
			wantCode:  "NotFound",
			wantError: "rpc error: code = NotFound desc = not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			logError(context.Background(), "unary", "server", "/test/Method", tt.err, nil, nil, defaultOptions())

			require.Len(t, handler.records, 1)
			record := handler.records[0]

			assert.Equal(t, tt.wantCode, record["code"])
			assert.Equal(t, tt.wantError, record["error"])
		})
	}
}

func TestLogError_WithRequestResponse(t *testing.T) {
	tests := []struct {
		name          string
		printRequest  bool
		printResponse bool
		req           interface{}
		resp          interface{}
		wantRequest   bool
		wantResponse  bool
	}{
		{
			name:          "打印请求和响应",
			printRequest:  true,
			printResponse: true,
			req:           map[string]string{"key": "value"},
			resp:          map[string]string{"result": "ok"},
			wantRequest:   true,
			wantResponse:  true,
		},
		{
			name:          "只打印请求",
			printRequest:  true,
			printResponse: false,
			req:           "test request",
			resp:          "test response",
			wantRequest:   true,
			wantResponse:  false,
		},
		{
			name:          "只打印响应",
			printRequest:  false,
			printResponse: true,
			req:           "test request",
			resp:          "test response",
			wantRequest:   false,
			wantResponse:  true,
		},
		{
			name:          "都不打印",
			printRequest:  false,
			printResponse: false,
			req:           "test request",
			resp:          "test response",
			wantRequest:   false,
			wantResponse:  false,
		},
		{
			name:          "nil请求即使开启也不打印",
			printRequest:  true,
			printResponse: true,
			req:           nil,
			resp:          "test response",
			wantRequest:   false,
			wantResponse:  true,
		},
		{
			name:          "nil响应即使开启也不打印",
			printRequest:  true,
			printResponse: true,
			req:           "test request",
			resp:          nil,
			wantRequest:   true,
			wantResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			opts := &options{
				PrintRequest:  tt.printRequest,
				PrintResponse: tt.printResponse,
			}
			logError(context.Background(), "unary", "server", "/test/Method", errors.New("error"), tt.req, tt.resp, opts)

			require.Len(t, handler.records, 1)
			record := handler.records[0]

			_, hasRequest := record["request"]
			_, hasResponse := record["response"]

			assert.Equal(t, tt.wantRequest, hasRequest)
			assert.Equal(t, tt.wantResponse, hasResponse)
		})
	}
}

func TestLogError_ContextCancellation(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		wantLevel string
	}{
		{
			name:   "正常上下文",
			ctx:    context.Background(),
			wantLevel: "ERROR",
		},
		{
			name:   "已取消上下文",
			ctx:    func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			wantLevel: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			logError(tt.ctx, "unary", "server", "/test/Method", errors.New("error"), nil, nil, defaultOptions())

			require.Len(t, handler.records, 1)
			assert.Equal(t, tt.wantLevel, handler.records[0]["level"])
		})
	}
}

func TestLogError_MultipleCalls(t *testing.T) {
	tests := []struct {
		name       string
		callCount  int
		wantRecords int
	}{
		{
			name:       "多次调用记录多条日志",
			callCount:  3,
			wantRecords: 3,
		},
		{
			name:       "单次调用记录单条日志",
			callCount:  1,
			wantRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			for i := 0; i < tt.callCount; i++ {
				logError(context.Background(), "unary", "server", "/test/Method", errors.New("error"), nil, nil, defaultOptions())
			}

			assert.Len(t, handler.records, tt.wantRecords)
		})
	}
}

func BenchmarkLogError(b *testing.B) {
	opts := defaultOptions()
	err := errors.New("benchmark error")
	ctx := context.Background()

	b.Run("without_request_response", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logError(ctx, "unary", "server", "/test/Method", err, nil, nil, opts)
		}
	})

	b.Run("with_request_response", func(b *testing.B) {
		req := map[string]string{"key": "value"}
		resp := map[string]string{"result": "ok"}
		optsWithPrint := &options{PrintRequest: true, PrintResponse: true}
		for i := 0; i < b.N; i++ {
			logError(ctx, "unary", "server", "/test/Method", err, req, resp, optsWithPrint)
		}
	})
}

func BenchmarkUnaryServerInterceptor(b *testing.B) {
	interceptor := UnaryServerInterceptor()
	handler := &mockUnaryHandler{resp: "response"}
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, "request", info, handler.handle)
	}
}

func BenchmarkUnaryClientInterceptor(b *testing.B) {
	interceptor := UnaryClientInterceptor()
	invoker := &mockInvoker{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = interceptor(ctx, "/test/Method", "request", "reply", nil, invoker.invoke)
	}
}

func TestLogOutput_Format(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	err := errors.New("format test error")
	logError(context.Background(), "unary", "client", "/test/FormatMethod", err, nil, nil, defaultOptions())

	output := buf.String()
	assert.Contains(t, output, "gRPC call error")
	assert.Contains(t, output, "rpc_type=unary")
	assert.Contains(t, output, "system=client")
	assert.Contains(t, output, "method=/test/FormatMethod")
	assert.Contains(t, output, "error=\"format test error\"")
	assert.Contains(t, output, "code=Unknown")
}

func TestLogOutput_WithRequestResponseText(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	req := map[string]interface{}{"id": 123, "name": "test"}
	resp := map[string]interface{}{"status": "ok"}
	opts := &options{PrintRequest: true, PrintResponse: true}

	err := errors.New("format test error")
	logError(context.Background(), "unary", "server", "/test/Method", err, req, resp, opts)

	output := buf.String()
	assert.Contains(t, output, "request=")
	assert.Contains(t, output, "response=")
}

func TestLogOutput_JsonHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	err := errors.New("json test error")
	logError(context.Background(), "stream", "server", "/test/Stream", err, nil, nil, defaultOptions())

	output := buf.String()
	assert.Contains(t, output, `"msg":"gRPC call error"`)
	assert.Contains(t, output, `"rpc_type":"stream"`)
	assert.Contains(t, output, `"system":"server"`)
	assert.Contains(t, output, `"method":"/test/Stream"`)
	assert.Contains(t, output, `"error":"json test error"`)
}

func TestLogError_VariousCodes(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want string
	}{
		{"OK", codes.OK, "OK"},
		{"Canceled", codes.Canceled, "Canceled"},
		{"Unknown", codes.Unknown, "Unknown"},
		{"InvalidArgument", codes.InvalidArgument, "InvalidArgument"},
		{"DeadlineExceeded", codes.DeadlineExceeded, "DeadlineExceeded"},
		{"NotFound", codes.NotFound, "NotFound"},
		{"AlreadyExists", codes.AlreadyExists, "AlreadyExists"},
		{"PermissionDenied", codes.PermissionDenied, "PermissionDenied"},
		{"ResourceExhausted", codes.ResourceExhausted, "ResourceExhausted"},
		{"FailedPrecondition", codes.FailedPrecondition, "FailedPrecondition"},
		{"Aborted", codes.Aborted, "Aborted"},
		{"OutOfRange", codes.OutOfRange, "OutOfRange"},
		{"Unimplemented", codes.Unimplemented, "Unimplemented"},
		{"Internal", codes.Internal, "Internal"},
		{"Unavailable", codes.Unavailable, "Unavailable"},
		{"DataLoss", codes.DataLoss, "DataLoss"},
		{"Unauthenticated", codes.Unauthenticated, "Unauthenticated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, restore := setupMockLogger()
			defer restore()

			err := status.Error(tt.code, "test error")
			logError(context.Background(), "unary", "server", "/test/Method", err, nil, nil, defaultOptions())

			require.Len(t, handler.records, 1)
			assert.Equal(t, tt.want, handler.records[0]["code"])
		})
	}
}

func TestInterceptors_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() (error, func())
		wantErrMsg string
	}{
		{
			name: "UnaryServer传播错误",
			setupFunc: func() (error, func()) {
				mockHandler := &mockUnaryHandler{err: errors.New("server error")}
				interceptor := UnaryServerInterceptor()
				_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, mockHandler.handle)
				return err, func() {}
			},
			wantErrMsg: "server error",
		},
		{
			name: "StreamServer传播错误",
			setupFunc: func() (error, func()) {
				mockHandler := &mockStreamHandler{err: errors.New("stream error")}
				interceptor := StreamServerInterceptor()
				err := interceptor(nil, &mockServerStream{}, &grpc.StreamServerInfo{FullMethod: "/test/Method"}, mockHandler.handle)
				return err, func() {}
			},
			wantErrMsg: "stream error",
		},
		{
			name: "UnaryClient传播错误",
			setupFunc: func() (error, func()) {
				mockInvoker := &mockInvoker{err: errors.New("client error")}
				interceptor := UnaryClientInterceptor()
				err := interceptor(context.Background(), "/test/Method", nil, nil, nil, mockInvoker.invoke)
				return err, func() {}
			},
			wantErrMsg: "client error",
		},
		{
			name: "StreamClient传播错误",
			setupFunc: func() (error, func()) {
				mockStreamer := &mockStreamer{err: errors.New("stream client error")}
				interceptor := StreamClientInterceptor()
				_, err := interceptor(context.Background(), nil, nil, "/test/Method", mockStreamer.stream)
				return err, func() {}
			},
			wantErrMsg: "stream client error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, cleanup := tt.setupFunc()
			defer cleanup()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestInterceptors_PreserveOriginalError(t *testing.T) {
	originalErr := status.Error(codes.NotFound, "original not found")

	tests := []struct {
		name      string
		getErr    func() error
		wantCode  codes.Code
		wantMsg   string
	}{
		{
			name: "UnaryServer保留原始错误",
			getErr: func() error {
				mockHandler := &mockUnaryHandler{err: originalErr}
				interceptor := UnaryServerInterceptor()
				_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, mockHandler.handle)
				return err
			},
			wantCode: codes.NotFound,
			wantMsg:  "original not found",
		},
		{
			name: "UnaryClient保留原始错误",
			getErr: func() error {
				mockInvoker := &mockInvoker{err: originalErr}
				interceptor := UnaryClientInterceptor()
				return interceptor(context.Background(), "/test/Method", nil, nil, nil, mockInvoker.invoke)
			},
			wantCode: codes.NotFound,
			wantMsg:  "original not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.getErr()
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
			assert.Equal(t, tt.wantMsg, st.Message())
		})
	}
}

func TestInterceptors_Concurrent(t *testing.T) {
	interceptor := UnaryServerInterceptor()
	handler := &mockUnaryHandler{err: errors.New("concurrent error")}
	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	ctx := context.Background()

	t.Run("并发调用拦截器", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Run("goroutine", func(t *testing.T) {
				t.Parallel()
				_, err := interceptor(ctx, "request", info, handler.handle)
				assert.Error(t, err)
			})
		}
	})
}

func TestUnaryClientInterceptor_WithCallOptions(t *testing.T) {
	mockInvoker := &mockInvoker{err: errors.New("error")}
	interceptor := UnaryClientInterceptor()
	
	var callOpts []grpc.CallOption
	err := interceptor(context.Background(), "/test/Method", nil, nil, nil, mockInvoker.invoke, callOpts...)
	
	assert.Error(t, err)
	assert.Equal(t, 1, mockInvoker.callCount)
}

func TestStreamClientInterceptor_ReturnsStream(t *testing.T) {
	type mockClientStream struct {
		grpc.ClientStream
	}
	
	mockStreamer := &mockStreamer{clientStream: &mockClientStream{}}
	interceptor := StreamClientInterceptor()
	
	stream, err := interceptor(context.Background(), nil, nil, "/test/Method", mockStreamer.stream)
	
	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, 1, mockStreamer.callCount)
}

func TestLogError_EmptyMethod(t *testing.T) {
	handler, restore := setupMockLogger()
	defer restore()

	logError(context.Background(), "unary", "server", "", errors.New("error"), nil, nil, defaultOptions())

	require.Len(t, handler.records, 1)
	assert.Equal(t, "", handler.records[0]["method"])
}

func TestLogError_LongErrorMessage(t *testing.T) {
	handler, restore := setupMockLogger()
	defer restore()

	longMsg := strings.Repeat("a", 10000)
	logError(context.Background(), "unary", "server", "/test/Method", errors.New(longMsg), nil, nil, defaultOptions())

	require.Len(t, handler.records, 1)
	assert.Equal(t, longMsg, handler.records[0]["error"])
}

func TestLogError_NilError(t *testing.T) {
	handler, restore := setupMockLogger()
	defer restore()

	logError(context.Background(), "unary", "server", "/test/Method", nil, nil, nil, defaultOptions())

	require.Len(t, handler.records, 1)
	assert.Nil(t, handler.records[0]["error"])
	assert.Equal(t, "OK", handler.records[0]["code"])
}
