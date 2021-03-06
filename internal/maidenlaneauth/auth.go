// Copyright 2019 SEE CONTRIBUTORS

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package maidenlanedauth

import (
	"context"

	"github.com/freight-trust/zeroxyz/internal/maidenlanederrors"
	"github.com/freight-trust/zeroxyz/pkg/maidenlanedplugins"
)

type maidenlanedContextKey int

const (
	maidenlanedContextKeySystemAuth maidenlanedContextKey = iota
	maidenlanedContextKeyAuthContext
	maidenlanedContextKeyAccessToken
)

var securityModule maidenlanedplugins.SecurityModule

// RegisterSecurityModule is the plug point to register a security module
func RegisterSecurityModule(sm maidenlanedplugins.SecurityModule) {
	securityModule = sm
}

// NewSystemAuthContext creates a system background context
func NewSystemAuthContext() context.Context {
	return context.WithValue(context.Background(), maidenlanedContextKeySystemAuth, true)
}

// IsSystemContext checks if a context was created as a system context
func IsSystemContext(ctx context.Context) bool {
	b, ok := ctx.Value(maidenlanedContextKeySystemAuth).(bool)
	return ok && b
}

// WithAuthContext adds an access token to a base context
func WithAuthContext(ctx context.Context, token string) (context.Context, error) {
	if securityModule != nil {
		ctxValue, err := securityModule.VerifyToken(token)
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, maidenlanedContextKeyAccessToken, token)
		ctx = context.WithValue(ctx, maidenlanedContextKeyAuthContext, ctxValue)
		return ctx, nil
	}
	return ctx, nil
}

// GetAuthContext extracts a previously stored auth context from the context
func GetAuthContext(ctx context.Context) interface{} {
	return ctx.Value(maidenlanedContextKeyAuthContext)
}

// GetAccessToken extracts a previously stored access token
func GetAccessToken(ctx context.Context) string {
	v, ok := ctx.Value(maidenlanedContextKeyAccessToken).(string)
	if ok {
		return v
	}
	return ""
}

// AuthRPC authorize an RPC call
func AuthRPC(ctx context.Context, method string, args ...interface{}) error {
	if securityModule != nil && !IsSystemContext(ctx) {
		authCtx := GetAuthContext(ctx)
		if authCtx == nil {
			return maidenlanederrors.Errorf(maidenlanederrors.SecurityModuleNoAuthContext)
		}
		return securityModule.AuthRPC(authCtx, method, args...)
	}
	return nil
}

// AuthRPCSubscribe authorize a subscribe RPC call
func AuthRPCSubscribe(ctx context.Context, namespace string, channel interface{}, args ...interface{}) error {
	if securityModule != nil && !IsSystemContext(ctx) {
		authCtx := GetAuthContext(ctx)
		if authCtx == nil {
			return maidenlanederrors.Errorf(maidenlanederrors.SecurityModuleNoAuthContext)
		}
		return securityModule.AuthRPCSubscribe(authCtx, namespace, channel, args...)
	}
	return nil
}

// AuthEventStreams authorize the whole of event streams
func AuthEventStreams(ctx context.Context) error {
	if securityModule != nil && !IsSystemContext(ctx) {
		authCtx := GetAuthContext(ctx)
		if authCtx == nil {
			return maidenlanederrors.Errorf(maidenlanederrors.SecurityModuleNoAuthContext)
		}
		return securityModule.AuthEventStreams(authCtx)
	}
	return nil
}

// AuthListAsyncReplies authorize the listing or searching of all replies
func AuthListAsyncReplies(ctx context.Context) error {
	if securityModule != nil && !IsSystemContext(ctx) {
		authCtx := GetAuthContext(ctx)
		if authCtx == nil {
			return maidenlanederrors.Errorf(maidenlanederrors.SecurityModuleNoAuthContext)
		}
		return securityModule.AuthListAsyncReplies(authCtx)
	}
	return nil
}

// AuthReadAsyncReplyByUUID authorize the query of an invidual reply by UUID
func AuthReadAsyncReplyByUUID(ctx context.Context) error {
	if securityModule != nil && !IsSystemContext(ctx) {
		authCtx := GetAuthContext(ctx)
		if authCtx == nil {
			return maidenlanederrors.Errorf(maidenlanederrors.SecurityModuleNoAuthContext)
		}
		return securityModule.AuthReadAsyncReplyByUUID(authCtx)
	}
	return nil
}
