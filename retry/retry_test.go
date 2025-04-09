// Copyright (C) 2021-2025 Chronicle Labs, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	tc := []struct {
		name       string
		attempts   int
		delay      time.Duration
		ctxTimeout time.Duration
		fn         func() func(context.Context) error
		wantFail   bool
		wantErr    error
	}{
		{
			name:       "try max attempts",
			attempts:   3,
			delay:      10 * time.Millisecond,
			ctxTimeout: 100 * time.Millisecond,
			fn: func() func(context.Context) error {
				return func(ctx context.Context) error {
					return errors.New("error")
				}
			},
			wantFail: true,
			wantErr:  errors.New("error"),
		},
		{
			name:       "try success",
			attempts:   3,
			delay:      10 * time.Millisecond,
			ctxTimeout: 100 * time.Millisecond,
			fn: func() func(context.Context) error {
				c := 0
				return func(ctx context.Context) error {
					c++
					if c == 3 {
						return nil
					}
					return errors.New("error")
				}
			},
		},
		{
			name:       "try fail",
			attempts:   3,
			delay:      10 * time.Millisecond,
			ctxTimeout: 100 * time.Millisecond,
			fn: func() func(context.Context) error {
				return func(ctx context.Context) error {
					return errors.New("error")
				}
			},
			wantFail: true,
			wantErr:  errors.New("error"),
		},
		{
			name:       "try ctx cancel",
			attempts:   -1,
			delay:      10 * time.Millisecond,
			ctxTimeout: 50 * time.Millisecond,
			fn: func() func(context.Context) error {
				return func(ctx context.Context) error {
					<-ctx.Done()
					return errors.New("error")
				}
			},
			wantFail: true,
			wantErr:  context.DeadlineExceeded,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			switch {
			case tt.wantFail:
				t.Run("try", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.False(t, Try(ctx, func(ctx context.Context) bool {
						return fn(ctx) == nil
					}, tt.attempts, tt.delay))
				})

				t.Run("try1", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, "", Try1(ctx, func(ctx context.Context) (string, bool) {
						if fn(ctx) == nil {
							return "success", true
						}
						return "", false
					}, tt.attempts, tt.delay))
				})

				t.Run("try2", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, []any{"", 0}, unpack(Try2(ctx, func(ctx context.Context) (string, int, bool) {
						if fn(ctx) == nil {
							return "success", 42, true
						}
						return "", 0, false
					}, tt.attempts, tt.delay)))
					assert.Equal(t, tt.wantErr, TryErr(ctx, fn, tt.attempts, tt.delay))
				})

				t.Run("tryErr", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, tt.wantErr, TryErr(ctx, func(ctx context.Context) error {
						return fn(ctx)
					}, tt.attempts, tt.delay))
				})

				t.Run("try1Err", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, []any{"", tt.wantErr}, unpack(Try1Err(ctx, func(ctx context.Context) (string, error) {
						if fn(ctx) == nil {
							return "success", nil
						}
						return "", errors.New("error")
					}, tt.attempts, tt.delay)))
				})

				t.Run("try2Err", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, []any{"", 0, tt.wantErr}, unpack(Try2Err(ctx, func(ctx context.Context) (string, int, error) {
						if fn(ctx) == nil {
							return "success", 42, nil
						}
						return "", 0, errors.New("error")
					}, tt.attempts, tt.delay)))
				})
			default:
				t.Run("try", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.True(t, Try(ctx, func(ctx context.Context) bool {
						return fn(ctx) == nil
					}, tt.attempts, tt.delay))
				})

				t.Run("try1", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, "success", Try1(ctx, func(ctx context.Context) (string, bool) {
						println("a")
						if fn(ctx) == nil {
							return "success", true
						}
						return "", false
					}, tt.attempts, tt.delay))
				})

				t.Run("try2", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, []any{"success", 42}, unpack(Try2(ctx, func(ctx context.Context) (string, int, bool) {
						if fn(ctx) == nil {
							return "success", 42, true
						}
						return "", 0, false
					}, tt.attempts, tt.delay)))
				})

				t.Run("tryErr", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Nil(t, TryErr(ctx, func(ctx context.Context) error {
						return fn(ctx)
					}, tt.attempts, tt.delay))
				})

				t.Run("try1Err", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, []any{"success", nil}, unpack(Try1Err(ctx, func(ctx context.Context) (string, error) {
						if fn(ctx) == nil {
							return "success", nil
						}
						return "", errors.New("error")
					}, tt.attempts, tt.delay)))
				})

				t.Run("try2Err", func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
					defer cancel()
					fn := tt.fn()
					assert.Equal(t, []any{"success", 42, nil}, unpack(Try2Err(ctx, func(ctx context.Context) (string, int, error) {
						if fn(ctx) == nil {
							return "success", 42, nil
						}
						return "", 0, errors.New("error")
					}, tt.attempts, tt.delay)))
				})
			}
		})
	}
}

func unpack(a ...any) []any {
	return a
}
