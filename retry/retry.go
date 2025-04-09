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
	"time"
)

const (
	TryAgain = false
	Stop     = true
)

// Try will call the function f until it returns true or the context is done.
// If attempts is negative, Try will try forever.
func Try(ctx context.Context, f func(context.Context) bool, attempts int, delay time.Duration) (ok bool) {
	for i := 0; attempts < 0 || i < attempts; i++ {
		if ctx.Err() != nil {
			return false
		}
		if f(ctx) {
			return true
		}
		if attempts < 0 || i < attempts {
			t := time.NewTimer(delay)
			select {
			case <-ctx.Done():
			case <-t.C:
			}
			t.Stop()
		}
	}
	return false
}

// Try1 is a helper function that simplifies the common case of retrying a
// function that returns a single value.
func Try1[T any](ctx context.Context, f func(context.Context) (T, bool), attempts int, delay time.Duration) (res T) {
	var ok bool
	ok = Try(ctx, func(ctx context.Context) bool {
		res, ok = f(ctx)
		return ok
	}, attempts, delay)
	return res
}

// Try2 is a helper function that simplifies the common case of retrying a
// function that returns two values.
func Try2[T1, T2 any](ctx context.Context, f func(context.Context) (T1, T2, bool), attempts int, delay time.Duration) (res1 T1, res2 T2) {
	var ok bool
	ok = Try(ctx, func(ctx context.Context) bool {
		res1, res2, ok = f(ctx)
		return ok
	}, attempts, delay)
	return res1, res2
}

// TryErr will call the function f until it returns no error or the context is
// done. If attempts is negative, TryErr will try forever.
func TryErr(ctx context.Context, f func(context.Context) error, attempts int, delay time.Duration) (err error) {
	Try(ctx, func(ctx context.Context) bool {
		err = f(ctx)
		return err == nil
	}, attempts, delay)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// Try1Err is a helper function that simplifies the common case of retrying a
// function that returns a single value and an error.
func Try1Err[T any](ctx context.Context, f func(context.Context) (T, error), attempts int, delay time.Duration) (res T, err error) {
	Try(ctx, func(ctx context.Context) bool {
		res, err = f(ctx)
		return err == nil
	}, attempts, delay)
	if ctx.Err() != nil {
		return res, ctx.Err()
	}
	return res, err
}

// Try2Err is a helper function that simplifies the common case of retrying a
// function that returns two values and an error.
func Try2Err[T1, T2 any](ctx context.Context, f func(context.Context) (T1, T2, error), attempts int, delay time.Duration) (res1 T1, res2 T2, err error) {
	Try(ctx, func(ctx context.Context) bool {
		res1, res2, err = f(ctx)
		return err == nil
	}, attempts, delay)
	if ctx.Err() != nil {
		return res1, res2, ctx.Err()
	}
	return res1, res2, err
}
