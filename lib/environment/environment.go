// Copyright 2016
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package environment

import "context"

type envKey string

// VersionKey is the context key under which the version of dim is stored
const VersionKey string = "dimVersion"

// StartTimeKey is the context key under which the server was created
const StartTimeKey string = "startTime"

// Set returns a new context with the new key/value pair
func Set(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, envKey(key), value)
}

// Get reads a given key from the context
func Get(ctx context.Context, key string) interface{} {
	return ctx.Value(envKey(key))
}
