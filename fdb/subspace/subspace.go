// FoundationDB Go Subspace Layer
// Copyright (c) 2013 FoundationDB, LLC

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package subspace provides an implementation of FoundationDB
// subspaces. Subspaces provide a way to define namespaces for different
// categories of data. As such, they are a basic technique of data modeling
// (https://foundationdb.com/documentation/data-modeling.html#subspaces-of-keys).
// Subspaces use the ordering of tuple elements defined by the FoundationDB
// tuple package to structure keyspace. As a best practice, API clients should
// use at least one subspace for application data.
package subspace

import (
	"github.com/FoundationDB/fdb-go/fdb"
	"github.com/FoundationDB/fdb-go/fdb/tuple"
	"bytes"
	"errors"
)

// Subspace represents a well-defined region of keyspace in a FoundationDB
// database.
type Subspace interface {
	// Sub returns a new Subspace whose prefix extends this Subspace with the
	// encoding of the provided element(s). These elements must comply with the
	// limitations on Tuple elements defined in the fdb.tuple package.
	Sub(el ...interface{}) Subspace

	// Bytes returns the literal bytes of the prefix of this subspace.
	Bytes() []byte

	// Pack returns the key encoding the specified Tuple with the prefix of this
	// Subspace prepended. The Tuple must comply with the limitations on Tuple
	// elements defined in the fdb.tuple package.
	Pack(t tuple.Tuple) fdb.Key

	// Unpack returns the Tuple encoded by the given key with the prefix of this
	// Subspace removed. Unpack will return an error if the key is not in this
	// Subspace or does not encode a well-formed Tuple.
	Unpack(k fdb.KeyConvertible) (tuple.Tuple, error)

	// Contains returns true if the provided key starts with the prefix of this
	// Subspace, indicating that the Subspace logically contains the key.
	Contains(k fdb.KeyConvertible) bool

	// All Subspaces implement fdb.KeyConvertible and may be used as
	// FoundationDB keys (corresponding to the prefix of this Subspace).
	fdb.KeyConvertible

	// All Subspaces implement fdb.ExactRange and may be used as ranges
	// (corresponding to all keys logically in this Subspace).
	fdb.ExactRange
}

type subspace struct {
	b []byte
}

// AllKeys returns the Subspace corresponding to all keys in a FoundationDB
// database.
func AllKeys() Subspace {
	return subspace{}
}

// FromTuple returns a new Subspace from the provided Tuple.
func FromTuple(t tuple.Tuple) Subspace {
	return subspace{t.Pack()}
}

// FromBytes returns a new Subspace from the provided bytes.
func FromBytes(b []byte) Subspace {
	s := make([]byte, len(b))
	copy(s, b)
	return subspace{b}
}

func (s subspace) Sub(el ...interface{}) Subspace {
	return subspace{concat(s.Bytes(), tuple.Tuple(el).Pack()...)}
}

func (s subspace) Bytes() []byte {
	return s.b
}

func (s subspace) Pack(t tuple.Tuple) fdb.Key {
	return fdb.Key(concat(s.b, t.Pack()...))
}

func (s subspace) Unpack(k fdb.KeyConvertible) (tuple.Tuple, error) {
	key := k.ToFDBKey()
	if !bytes.HasPrefix(key, s.b) {
		return nil, errors.New("key is not in subspace")
	}
	return tuple.Unpack(key[len(s.b):])
}

func (s subspace) Contains(k fdb.KeyConvertible) bool {
	return bytes.HasPrefix(k.ToFDBKey(), s.b)
}

func (s subspace) ToFDBKey() fdb.Key {
	return fdb.Key(s.b)
}

func (s subspace) BeginKey() fdb.Key {
	return concat(s.b, 0x00)
}

func (s subspace) EndKey() fdb.Key {
	return concat(s.b, 0xFF)
}

func (s subspace) BeginKeySelector() fdb.KeySelector {
	return fdb.FirstGreaterOrEqual(s.BeginKey())
}

func (s subspace) EndKeySelector() fdb.KeySelector {
	return fdb.FirstGreaterOrEqual(s.EndKey())
}

func concat(a []byte, b ...byte) []byte {
	r := make([]byte, len(a) + len(b))
	copy(r, a)
	copy(r[len(a):], b)
	return r
}
