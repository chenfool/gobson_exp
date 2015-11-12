// Copyright 2015 David Li
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package bson

import (
	"math"
	"unsafe"
)

type ByteOrder interface {
	AppendInt32([]byte, int32) []byte
	AppendInt64([]byte, int64) []byte
	AppendFloat64([]byte, float64) []byte
	SetInt32([]byte, int, int32)
	IsBigEndian() bool
	IsLittleEndian() bool
}

type littleEndian struct{}
type bigEndian struct{}

var LittleEndian littleEndian
var BigEndian bigEndian
var byteOrder ByteOrder

func init() {
	const N int = int(unsafe.Sizeof(0))
	x := 0x1234
	p := unsafe.Pointer(&x)
	p2 := (*[N]byte)(p)
	if p2[0] == 0 {
		byteOrder = BigEndian
	} else {
		byteOrder = LittleEndian
	}
}

func GetByteOrder() ByteOrder {
	return byteOrder
}

func IsLittleEndian() bool {
	return byteOrder == LittleEndian
}

func IsBigEndian() bool {
	return byteOrder == BigEndian
}

func (le littleEndian) AppendInt32(b []byte, v int32) []byte {
	u := uint32(v)
	return append(b, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}

func (le littleEndian) AppendInt64(b []byte, v int64) []byte {
	u := uint64(v)
	return append(b, byte(u), byte(u>>8), byte(u>>16), byte(u>>24),
		byte(u>>32), byte(u>>40), byte(u>>48), byte(u>>56))
}

func (le littleEndian) AppendFloat64(b []byte, v float64) []byte {
	u := int64(math.Float64bits(v))
	return le.AppendInt64(b, u)
}

func (le littleEndian) SetInt32(b []byte, pos int, v int32) {
	b[pos+0] = byte(v)
	b[pos+1] = byte(v >> 8)
	b[pos+2] = byte(v >> 16)
	b[pos+3] = byte(v >> 24)
}

func (le littleEndian) IsBigEndian() bool {
	return false
}

func (le littleEndian) IsLittleEndian() bool {
	return true
}

func (be bigEndian) AppendInt32(b []byte, v int32) []byte {
	u := uint32(v)
	return append(b, byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
}

func (be bigEndian) AppendInt64(b []byte, v int64) []byte {
	u := uint64(v)
	return append(b, byte(u>>56), byte(u>>48), byte(u>>40), byte(u>>32),
		byte(u>>24), byte(u>>16), byte(u>>8), byte(u))
}

func (be bigEndian) AppendFloat64(b []byte, v float64) []byte {
	u := int64(math.Float64bits(v))
	return be.AppendInt64(b, u)
}

func (be bigEndian) SetInt32(b []byte, pos int, v int32) {
	b[pos+0] = byte(v >> 24)
	b[pos+1] = byte(v >> 16)
	b[pos+2] = byte(v >> 8)
	b[pos+3] = byte(v)
}

func (be bigEndian) IsBigEndian() bool {
	return true
}

func (be bigEndian) IsLittleEndian() bool {
	return false
}