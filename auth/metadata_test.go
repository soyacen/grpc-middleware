package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthFromMD_Success(t *testing.T) {
	tests := []struct {
		name           string
		md             metadata.MD
		expectedScheme string
		wantToken      string
		wantErr        error
	}{
		{
			name:           "正常提取Bearer token",
			md:             metadata.Pairs("authorization", "Bearer valid_token_123"),
			expectedScheme: "Bearer",
			wantToken:      "valid_token_123",
			wantErr:        nil,
		},
		{
			name:           "正常提取Basic token",
			md:             metadata.Pairs("authorization", "Basic dXNlcjpwYXNz"),
			expectedScheme: "Basic",
			wantToken:      "dXNlcjpwYXNz",
			wantErr:        nil,
		},
		{
			name:           "scheme大小写不敏感",
			md:             metadata.Pairs("authorization", "bearer lowercase_token"),
			expectedScheme: "Bearer",
			wantToken:      "lowercase_token",
			wantErr:        nil,
		},
		{
			name:           "scheme为大写",
			md:             metadata.Pairs("authorization", "BEARER uppercase_token"),
			expectedScheme: "Bearer",
			wantToken:      "uppercase_token",
			wantErr:        nil,
		},
		{
			name:           "token包含空格",
			md:             metadata.Pairs("authorization", "Bearer token with spaces"),
			expectedScheme: "Bearer",
			wantToken:      "token with spaces",
			wantErr:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), tt.md)
			token, err := AuthFromMD(ctx, tt.expectedScheme)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

func TestAuthFromMD_MissingHeader(t *testing.T) {
	tests := []struct {
		name           string
		md             metadata.MD
		expectedScheme string
		wantErrCode    codes.Code
		wantErrMsg     string
	}{
		{
			name:           "缺少authorization头",
			md:             metadata.MD{},
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Request unauthenticated with Bearer",
		},
		{
			name:           "authorization头为空切片",
			md:             metadata.Pairs("authorization", ""),
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Bad authorization string",
		},
		{
			name:           "其他头存在但缺少authorization",
			md:             metadata.Pairs("x-request-id", "12345"),
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Request unauthenticated with Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), tt.md)
			token, err := AuthFromMD(ctx, tt.expectedScheme)

			assert.Empty(t, token)
			assert.Error(t, err)
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, tt.wantErrCode, st.Code())
			assert.Equal(t, tt.wantErrMsg, st.Message())
		})
	}
}

func TestAuthFromMD_InvalidFormat(t *testing.T) {
	tests := []struct {
		name           string
		md             metadata.MD
		expectedScheme string
		wantErrCode    codes.Code
		wantErrMsg     string
	}{
		{
			name:           "缺少空格分隔符",
			md:             metadata.Pairs("authorization", "BearerToken"),
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Bad authorization string",
		},
		{
			name:           "只有scheme没有token",
			md:             metadata.Pairs("authorization", "Bearer"),
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Bad authorization string",
		},
		{
			name:           "错误的scheme",
			md:             metadata.Pairs("authorization", "Basic token123"),
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Request unauthenticated with Bearer",
		},
		{
			name:           "大小写不同的错误scheme",
			md:             metadata.Pairs("authorization", "basic token123"),
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Request unauthenticated with Bearer",
		},
		{
			name:           "空的authorization值",
			md:             metadata.MD{"authorization": []string{""}},
			expectedScheme: "Bearer",
			wantErrCode:    codes.Unauthenticated,
			wantErrMsg:     "Bad authorization string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), tt.md)
			token, err := AuthFromMD(ctx, tt.expectedScheme)

			assert.Empty(t, token)
			assert.Error(t, err)
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, tt.wantErrCode, st.Code())
			assert.Equal(t, tt.wantErrMsg, st.Message())
		})
	}
}

func TestAuthFromMD_MultipleValues(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"Bearer first_token", "Bearer second_token"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	token, err := AuthFromMD(ctx, "Bearer")

	assert.NoError(t, err)
	assert.Equal(t, "first_token", token)
}

func TestAuthFromMD_DifferentSchemes(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		expectedScheme string
		wantToken      string
		wantErr        bool
	}{
		{"Bearer方案", "Bearer token123", "Bearer", "token123", false},
		{"Basic方案", "Basic dXNlcjpwYXNz", "Basic", "dXNlcjpwYXNz", false},
		{"Digest方案", "Digest username=\"user\"", "Digest", "username=\"user\"", false},
		{"自定义方案", "ApiKey key123", "ApiKey", "key123", false},
		{"错误方案", "Bearer token123", "Basic", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.Pairs("authorization", tt.value)
			ctx := metadata.NewIncomingContext(context.Background(), md)
			token, err := AuthFromMD(ctx, tt.expectedScheme)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}
