package errorlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name              string
		wantPrintRequest  bool
		wantPrintResponse bool
	}{
		{
			name:              "默认值",
			wantPrintRequest:  false,
			wantPrintResponse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultOptions()
			assert.Equal(t, tt.wantPrintRequest, got.PrintRequest)
			assert.Equal(t, tt.wantPrintResponse, got.PrintResponse)
		})
	}
}

func TestWithPrintRequest(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
		want   bool
	}{
		{"启用", true, true},
		{"禁用", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(WithPrintRequest(tt.enable))
			assert.Equal(t, tt.want, o.PrintRequest)
			assert.False(t, o.PrintResponse)
		})
	}
}

func TestWithPrintResponse(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
		want   bool
	}{
		{"启用", true, true},
		{"禁用", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(WithPrintResponse(tt.enable))
			assert.Equal(t, tt.want, o.PrintResponse)
			assert.False(t, o.PrintRequest)
		})
	}
}

func TestApply_MultipleOptions(t *testing.T) {
	tests := []struct {
		name              string
		opts              []Option
		wantPrintRequest  bool
		wantPrintResponse bool
	}{
		{
			name:              "全部启用",
			opts:              []Option{WithPrintRequest(true), WithPrintResponse(true)},
			wantPrintRequest:  true,
			wantPrintResponse: true,
		},
		{
			name:              "全部禁用",
			opts:              []Option{WithPrintRequest(false), WithPrintResponse(false)},
			wantPrintRequest:  false,
			wantPrintResponse: false,
		},
		{
			name:              "启用请求禁用响应",
			opts:              []Option{WithPrintRequest(true), WithPrintResponse(false)},
			wantPrintRequest:  true,
			wantPrintResponse: false,
		},
		{
			name:              "禁用请求启用响应",
			opts:              []Option{WithPrintRequest(false), WithPrintResponse(true)},
			wantPrintRequest:  false,
			wantPrintResponse: true,
		},
		{
			name:              "反向顺序应用",
			opts:              []Option{WithPrintResponse(true), WithPrintRequest(true)},
			wantPrintRequest:  true,
			wantPrintResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			assert.Equal(t, tt.wantPrintRequest, o.PrintRequest)
			assert.Equal(t, tt.wantPrintResponse, o.PrintResponse)
		})
	}
}

func TestApply_Empty(t *testing.T) {
	tests := []struct {
		name              string
		opts              []Option
		wantPrintRequest  bool
		wantPrintResponse bool
	}{
		{
			name:              "空选项保持默认值",
			opts:              []Option{},
			wantPrintRequest:  false,
			wantPrintResponse: false,
		},
		{
			name:              "nil选项保持默认值",
			opts:              nil,
			wantPrintRequest:  false,
			wantPrintResponse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			assert.Equal(t, tt.wantPrintRequest, o.PrintRequest)
			assert.Equal(t, tt.wantPrintResponse, o.PrintResponse)
		})
	}
}

func TestApply_Override(t *testing.T) {
	tests := []struct {
		name              string
		opts              []Option
		wantPrintRequest  bool
		wantPrintResponse bool
	}{
		{
			name:              "后覆盖前",
			opts:              []Option{WithPrintRequest(true), WithPrintRequest(false)},
			wantPrintRequest:  false,
			wantPrintResponse: false,
		},
		{
			name:              "先false后true",
			opts:              []Option{WithPrintRequest(false), WithPrintRequest(true)},
			wantPrintRequest:  true,
			wantPrintResponse: false,
		},
		{
			name:              "先true后false再true",
			opts:              []Option{WithPrintRequest(true), WithPrintRequest(false), WithPrintRequest(true)},
			wantPrintRequest:  true,
			wantPrintResponse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			assert.Equal(t, tt.wantPrintRequest, o.PrintRequest)
			assert.Equal(t, tt.wantPrintResponse, o.PrintResponse)
		})
	}
}

func TestOptionsStruct_DirectAccess(t *testing.T) {
	tests := []struct {
		name              string
		printRequest      bool
		printResponse     bool
		wantPrintRequest  bool
		wantPrintResponse bool
	}{
		{"true_true", true, true, true, true},
		{"true_false", true, false, true, false},
		{"false_true", false, true, false, true},
		{"false_false", false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{
				PrintRequest:  tt.printRequest,
				PrintResponse: tt.printResponse,
			}
			assert.Equal(t, tt.wantPrintRequest, o.PrintRequest)
			assert.Equal(t, tt.wantPrintResponse, o.PrintResponse)
		})
	}
}

func TestApply_SingleOption(t *testing.T) {
	tests := []struct {
		name              string
		opt               Option
		wantPrintRequest  bool
		wantPrintResponse bool
	}{
		{
			name:              "只启用PrintRequest",
			opt:               WithPrintRequest(true),
			wantPrintRequest:  true,
			wantPrintResponse: false,
		},
		{
			name:              "只启用PrintResponse",
			opt:               WithPrintResponse(true),
			wantPrintRequest:  false,
			wantPrintResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opt)
			assert.Equal(t, tt.wantPrintRequest, o.PrintRequest)
			assert.Equal(t, tt.wantPrintResponse, o.PrintResponse)
		})
	}
}

func BenchmarkDefaultOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = defaultOptions()
	}
}

func BenchmarkApplyOptions(b *testing.B) {
	opts := []Option{
		WithPrintRequest(true),
		WithPrintResponse(true),
	}

	for i := 0; i < b.N; i++ {
		o := defaultOptions()
		o.apply(opts...)
		_ = o
	}
}

func BenchmarkWithPrintRequest(b *testing.B) {
	opt := WithPrintRequest(true)

	for i := 0; i < b.N; i++ {
		o := defaultOptions()
		o.apply(opt)
		_ = o
	}
}

func BenchmarkWithPrintResponse(b *testing.B) {
	opt := WithPrintResponse(true)

	for i := 0; i < b.N; i++ {
		o := defaultOptions()
		o.apply(opt)
		_ = o
	}
}
