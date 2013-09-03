// FoundationDB Go API
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

package fdb

/*
 #cgo LDFLAGS: -lfdb_c -lm
 #define FDB_API_VERSION 100
 #include <foundationdb/fdb_c.h>
 #include <string.h>

 extern void notifyChannel(void*);

 void go_callback(FDBFuture* f, void* ch) {
     notifyChannel(ch);
 }

 void go_set_callback(void* f, void* ch) {
     fdb_future_set_callback(f, (FDBCallback)&go_callback, ch);
 }
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type future struct {
	f *C.FDBFuture
}

func fdb_future_block_until_ready(f *C.FDBFuture) {
	if C.fdb_future_is_ready(f) != 0 {
		return
	}

	ch := make(chan struct{})
	C.go_set_callback(unsafe.Pointer(f), unsafe.Pointer(&ch))
	<-ch
}

func (f *future) BlockUntilReady() {
	fdb_future_block_until_ready(f.f)
}

func (f *future) IsReady() bool {
	return C.fdb_future_is_ready(f.f) != 0
}

func (f *future) Cancel() {
	C.fdb_future_cancel(f.f)
}

type FutureValue struct {
	future
	v   []byte
	set bool
}

func (v *FutureValue) destroy() {
	C.fdb_future_destroy(v.f)
}

func (v *FutureValue) GetWithError() ([]byte, error) {
	if v.set {
		return v.v, nil
	}

	v.BlockUntilReady()
	var present C.fdb_bool_t
	var value *C.uint8_t
	var length C.int
	if err := C.fdb_future_get_value(v.f, &present, &value, &length); err != 0 {
		if err != 2017 {
			return nil, Error{Code: err}
		}
	}
	if present != 0 {
		v.v = C.GoBytes(unsafe.Pointer(value), length)
	}
	v.set = true
	C.fdb_future_release_memory(v.f)
	return v.v, nil
}

func (v *FutureValue) GetOrPanic() []byte {
	val, err := v.GetWithError()
	if err != nil {
		panic(err)
	}
	return val
}

type FutureKey struct {
	future
	k []byte
}

func (k *FutureKey) destroy() {
	C.fdb_future_destroy(k.f)
}

func (k *FutureKey) GetWithError() ([]byte, error) {
	if k.k != nil {
		return k.k, nil
	}

	k.BlockUntilReady()
	var value *C.uint8_t
	var length C.int
	if err := C.fdb_future_get_key(k.f, &value, &length); err != 0 {
		if err != 2017 {
			return nil, Error{Code: err}
		}
	}
	k.k = C.GoBytes(unsafe.Pointer(value), length)
	C.fdb_future_release_memory(k.f)
	return k.k, nil
}

func (k *FutureKey) GetOrPanic() []byte {
	val, err := k.GetWithError()
	if err != nil {
		panic(err)
	}
	return val
}

type FutureNil struct {
	future
}

func makeFutureNil(f *C.FDBFuture) *FutureNil {
	ret := &FutureNil{future: future{f: f}}
	runtime.SetFinalizer(ret, (*FutureNil).destroy)
	return ret
}

func (f *FutureNil) destroy() {
	C.fdb_future_destroy(f.f)
}

func (f *FutureNil) GetWithError() error {
	fdb_future_block_until_ready(f.f)
	if err := C.fdb_future_get_error(f.f); err != 0 {
		return Error{Code: err}
	}
	return nil
}

func (f *FutureNil) GetOrPanic() {
	if err := f.GetWithError(); err != nil {
		panic(err)
	}
}

type FutureKeyValueArray struct {
	future
}

func (f *FutureKeyValueArray) destroy() {
	C.fdb_future_destroy(f.f)
}

func stringRefToSlice(ptr uintptr) []byte {
	size := int(*((*C.int)(unsafe.Pointer(ptr+8))))

	if size == 0 {
		return []byte{}
	}

	ret := make([]byte, size)

	dst := unsafe.Pointer(&(ret[0]))
	src := unsafe.Pointer(*(**C.uint8_t)(unsafe.Pointer(ptr)))

	C.memcpy(dst, src, C.size_t(size))

	return ret
}

func (f *FutureKeyValueArray) GetWithError() ([]KeyValue, bool, error) {
	f.BlockUntilReady()

	var kvs *C.void
	var count C.int
	var more C.fdb_bool_t

	if err := C.fdb_future_get_keyvalue_array(f.f, (**C.FDBKeyValue)(unsafe.Pointer(&kvs)), &count, &more); err != 0 {
		return nil, false, Error{Code: err}
	}

	ret := make([]KeyValue, int(count))

	for i := 0; i < int(count); i++ {
		kvptr := uintptr(unsafe.Pointer(kvs)) + uintptr(i * 24)

		ret[i].Key = stringRefToSlice(kvptr)
		ret[i].Value = stringRefToSlice(kvptr + 12)

	}

 	return ret, (more != 0), nil
}

func (f *FutureKeyValueArray) GetOrPanic() ([]KeyValue, bool) {
	kvs, more, err := f.GetWithError()
	if err != nil {
		panic(err)
	}
	return kvs, more
}

type FutureVersion struct {
	future
}

func (v *FutureVersion) destroy() {
	C.fdb_future_destroy(v.f)
}

func (v *FutureVersion) GetWithError() (int64, error) {
	v.BlockUntilReady()
	var ver C.int64_t
	if err := C.fdb_future_get_version(v.f, &ver); err != 0 {
		return 0, Error{Code: err}
	}
	return int64(ver), nil
}

func (v *FutureVersion) GetOrPanic() int64 {
	val, err := v.GetWithError()
	if err != nil {
		panic(err)
	}
	return val
}
