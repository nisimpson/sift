package jsonapi

import (
	"context"
	"net/url"

	"github.com/nisimpson/sift"
)

// Wrapper wraps another [sift.Adapter] to enable JSON:API query parameter support.
// It parses JSON:API filters and evaluates them using the wrapped adapter.
type Wrapper struct {
	adapter sift.Adapter
	jsonapi *Adapter
}

// Wrap creates a new wrapper around the given [sift.Adapter].
// This allows parsing JSON:API query parameters and evaluating them
// with any backend adapter (DynamoDB, expr-lang, SQL, etc.).
//
// Example:
//
//	dynamoAdapter := dynamodb.NewAdapter()
//	wrapped := jsonapi.Wrap(dynamoAdapter)
//	err := wrapped.ParseThru(ctx, queryParams)
//	expr, _ := dynamoAdapter.Expression()
func Wrap(adapter sift.Adapter) *Wrapper {
	return &Wrapper{
		adapter: adapter,
		jsonapi: NewAdapter(),
	}
}

// ParseThru parses JSON:API query parameters and evaluates them using the wrapped adapter.
// This is a convenience method that combines [Adapter.Parse]() and [sift.Thru]().
func (w *Wrapper) ParseThru(ctx context.Context, values url.Values) error {
	// Parse JSON:API to Sift expression
	filter, err := w.jsonapi.Parse(values)
	if err != nil {
		return err
	}

	// Evaluate with wrapped adapter
	return sift.Thru(ctx, w.adapter, filter)
}

// Adapter returns the wrapped adapter.
// Use this to access the underlying adapter's methods after [Wrapper.ParseThru]().
func (w *Wrapper) Adapter() sift.Adapter {
	return w.adapter
}

// JSONAPIAdapter returns the JSON:API adapter.
// Use this to access JSON:API-specific methods like [Adapter.Query]() and [Adapter.Parameters]().
func (w *Wrapper) JSONAPIAdapter() *Adapter {
	return w.jsonapi
}
