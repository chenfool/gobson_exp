// Copyright 2015-2016 David Li
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
	"bytes"
	"fmt"
	"math"
	"reflect"
)

type Bson struct {
	raw      []byte
	offset   int // start position in raw
	child    *Bson
	inChild  bool
	finished bool
}

const initialBufferSize = 64
const eod = byte(0x00) // end of doc

func (bson *Bson) reserveInt32() (pos int) {
	pos = len(bson.raw)
	bson.raw = append(bson.raw, 0, 0, 0, 0)
	return pos
}

func (bson *Bson) setInt32(pos int, v int32) {
	bson.raw[pos+0] = byte(v)
	bson.raw[pos+1] = byte(v >> 8)
	bson.raw[pos+2] = byte(v >> 16)
	bson.raw[pos+3] = byte(v >> 24)
}

func (bson *Bson) appendType(t BsonType)  {
	bson.raw = append(bson.raw, byte(t))
}

func (bson *Bson) appendCString(v string) {
	const eos byte = 0x00 // end of cstring
	bson.raw = append(bson.raw, []byte(v)...)
	bson.raw = append(bson.raw, eos)
}

func (bson *Bson) appendBytes(v ...byte)  {
	bson.raw = append(bson.raw, v...)
}

func (bson *Bson) appendInt32(v int32) {
	u := uint32(v)
	bson.raw = append(bson.raw, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}

func (bson *Bson) appendInt64(v int64) {
	u := uint64(v)
	bson.raw = append(bson.raw, byte(u), byte(u>>8), byte(u>>16), byte(u>>24),
		byte(u>>32), byte(u>>40), byte(u>>48), byte(u>>56))
}

func (bson *Bson) appendFloat64(v float64) {
	u := int64(math.Float64bits(v))
	bson.appendInt64(u)
}

func NewBson() *Bson {
	bson := &Bson{raw: make([]byte, 0, initialBufferSize)}
	bson.reserveInt32()
	return bson
}

func NewBsonWithRaw(raw []byte) *Bson {
	return &Bson{raw: raw, finished: true}
}

func (bson *Bson) Validate() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid bson: %v", r)
		}
	}()

	if !bson.finished {
		return fmt.Errorf("unfinished bson")
	}

	if bson.Length() != len(bson.Raw()) {
		return fmt.Errorf("invalid bson length")
	}

	it := bson.Iterator()
	for it.Next() {
		it.Name()
		it.Value()
	}

	return nil
}

func (bson *Bson) checkBeforeAppend() {
	if bson.finished {
		panic("the bson is finished")
	}

	if bson.inChild {
		panic("in child bson")
	}
}

func (bson *Bson) Finish() {
	bson.checkBeforeAppend()
	bson.raw = append(bson.raw, eod)
	bson.setInt32(bson.offset, int32(len(bson.raw)-bson.offset))
	bson.finished = true
}

func (bson *Bson) Raw() []byte {
	return bson.raw[bson.offset:]
}

func (bson *Bson) AppendFloat64(name string, value float64) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeFloat64)
	bson.appendCString(name)
	bson.appendFloat64(value)
}

func (bson *Bson) AppendString(name string, value string) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeString)
	bson.appendCString(name)
	bson.appendInt32(int32(len(value)+1))
	bson.appendCString(value)
}

func (bson *Bson) AppendBson(name string, value *Bson) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeBson)
	bson.appendCString(name)
	bson.appendBytes(value.Raw()...)
}

func (bson *Bson) AppendBsonStart(name string) (child *Bson) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeBson)
	bson.appendCString(name)
	child = &Bson{raw: bson.raw, offset: len(bson.raw)}
	child.reserveInt32()
	bson.inChild = true
	bson.child = child
	return child
}

func (bson *Bson) AppendBsonEnd() {
	if !bson.inChild {
		panic("not in child bson")
	}
	if bson.finished {
		panic("the bson is finished")
	}
	if bson.child.raw[len(bson.child.raw)-1] != eod {
		panic("the child bson is not finished")
	}
	bson.raw = bson.child.raw
	bson.child = nil
	bson.inChild = false
}

func (bson *Bson) AppendArray(name string, value *BsonArray) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeArray)
	bson.appendCString(name)
	bson.appendBytes(value.Raw()...)
}

func (bson *Bson) AppendArrayStart(name string) (child *BsonArray) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeArray)
	bson.appendCString(name)
	child = &BsonArray{bson: Bson{raw: bson.raw, offset: len(bson.raw)}}
	child.bson.reserveInt32()
	bson.inChild = true
	bson.child = &child.bson
	return child
}

func (bson *Bson) AppendArrayEnd() {
	if !bson.inChild {
		panic("not in child array")
	}
	if bson.finished {
		panic("the bson is finished")
	}
	if bson.child.raw[len(bson.child.raw)-1] != eod {
		panic("the child array is not finished")
	}
	bson.raw = bson.child.raw
	bson.child = nil
	bson.inChild = false
}

func (bson *Bson) AppendBinary(name string, value Binary) {
	bson.checkBeforeAppend()
	if value.Data == nil {
		panic("binary is null")
	}
	bson.appendType(BsonTypeBinary)
	bson.appendCString(name)
	bson.appendInt32(int32(len(value.Data)))
	bson.appendBytes(byte(value.Subtype))
	bson.appendBytes(value.Data...)
}

func (bson *Bson) AppendObjectId(name string, value ObjectId) {
	bson.checkBeforeAppend()
	if !value.IsValid() {
		panic(fmt.Sprintf("invalid ObjectId: %s", value))
	}
	bson.appendType(BsonTypeObjectId)
	bson.appendCString(name)
	bson.appendBytes([]byte(value)...)
}

func (bson *Bson) AppendBool(name string, value bool) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeBool)
	bson.appendCString(name)
	if value {
		bson.appendBytes(byte(1))
	} else {
		bson.appendBytes(byte(0))
	}
}

func (bson *Bson) AppendDate(name string, value Date) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeDate)
	bson.appendCString(name)
	bson.appendInt64(int64(value))
}

func (bson *Bson) AppendNull(name string) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeNull)
	bson.appendCString(name)
}

func (bson *Bson) AppendRegex(name string, value RegEx) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeRegEx)
	bson.appendCString(name)
	bson.appendCString(value.Pattern)
	bson.appendCString(value.Options)
}

func (bson *Bson) AppendInt32(name string, value int32) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeInt32)
	bson.appendCString(name)
	bson.appendInt32(value)
}

func (bson *Bson) AppendTimestamp(name string, value Timestamp) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeTimestamp)
	bson.appendCString(name)
	bson.appendInt32(value.Increment)
	bson.appendInt32(value.Second)
}

func (bson *Bson) AppendInt64(name string, value int64) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeInt64)
	bson.appendCString(name)
	bson.appendInt64(value)
}

func (bson *Bson) AppendMinKey(name string) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeMinKey)
	bson.appendCString(name)
}

func (bson *Bson) AppendMaxKey(name string) {
	bson.checkBeforeAppend()
	bson.appendType(BsonTypeMaxKey)
	bson.appendCString(name)
}

func (bson *Bson) Append(name string, value interface{}) {
	switch value.(type) {
	case float32:
		bson.AppendFloat64(name, float64(value.(float32)))
	case float64:
		bson.AppendFloat64(name, value.(float64))
	case int8:
		bson.AppendInt32(name, int32(value.(int8)))
	case int16:
		bson.AppendInt32(name, int32(value.(int16)))
	case int32:
		bson.AppendInt32(name, value.(int32))
	case int64:
		val := value.(int64)
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			bson.AppendInt32(name, int32(val))
		} else {
			bson.AppendInt64(name, int64(val))
		}
	case uint8:
		bson.AppendInt32(name, int32(value.(uint8)))
	case uint16:
		bson.AppendInt32(name, int32(value.(uint16)))
	case uint32:
		val := value.(uint32)
		if int32(val) < 0 {
			bson.AppendInt64(name, int64(val))
		} else {
			bson.AppendInt32(name, int32(val))
		}
	case uint64:
		val := int64(value.(uint64))
		if val < 0 {
			panic("bson has no uint64 type, and value is too large to fit correctly in an int64")
		}
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			bson.AppendInt32(name, int32(val))
		} else {
			bson.AppendInt64(name, int64(val))
		}
	case int:
		val := int64(value.(int))
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			bson.AppendInt32(name, int32(val))
		} else {
			bson.AppendInt64(name, int64(val))
		}
	case uint:
		val := int64(value.(uint))
		if val < 0 {
			panic("bson has no uint64 type, and value is too large to fit correctly in an int64")
		}
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			bson.AppendInt32(name, int32(val))
		} else {
			bson.AppendInt64(name, int64(val))
		}
	case uintptr:
		val := int64(value.(uintptr))
		if val < 0 {
			panic("bson has no uint64 type, and value is too large to fit correctly in an int64")
		}
		if val >= math.MinInt32 && val <= math.MaxInt32 {
			bson.AppendInt32(name, int32(val))
		} else {
			bson.AppendInt64(name, int64(val))
		}
	case bool:
		bson.AppendBool(name, value.(bool))
	case string:
		bson.AppendString(name, value.(string))
	case nil:
		bson.AppendNull(name)
	case ObjectId:
		bson.AppendObjectId(name, value.(ObjectId))
	case Date:
		bson.AppendDate(name, value.(Date))
	case RegEx:
		bson.AppendRegex(name, value.(RegEx))
	case Timestamp:
		bson.AppendTimestamp(name, value.(Timestamp))
	case Binary:
		bson.AppendBinary(name, value.(Binary))
	case orderKey:
		val := value.(orderKey)
		if val == MaxKey {
			bson.AppendMaxKey(name)
		} else if val == MinKey {
			bson.AppendMinKey(name)
		} else {
			panic("invalid orderkey")
		}
	case Map:
		m := value.(Map)
		child := bson.AppendBsonStart(name)
		m.toBson(child)
		child.Finish()
		bson.AppendBsonEnd()
	case Doc:
		d := value.(Doc)
		child := bson.AppendBsonStart(name)
		d.toBson(child)
		child.Finish()
		bson.AppendBsonEnd()
	default:
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			l := v.Len()
			child := bson.AppendArrayStart(name)
			for i := 0; i < l; i++ {
				child.Append(v.Index(i).Interface())
			}
			child.Finish()
			bson.AppendArrayEnd()
			return
		case reflect.Map:
			child := bson.AppendBsonStart(name)
			for _, k := range v.MapKeys() {
				child.Append(k.String(), v.MapIndex(k).Interface())
			}
			child.Finish()
			bson.AppendBsonEnd()
			return
		case reflect.Ptr:
			if v.IsNil() {
				bson.AppendNull(name)
			} else {
				bson.Append(name, v.Elem().Interface())
			}
			return
		case reflect.Struct:
			child := bson.AppendBsonStart(name)
			structToBson(v, child)
			child.Finish()
			bson.AppendBsonEnd()
			return
		}
		// Complex64, Complex128
		// Chan, Func
		// UnsafePointer
		panic(fmt.Errorf("can't append %s(%v) to bson", reflect.TypeOf(value).String(), value))
	}
}

func (bson *Bson) Iterator() *BsonIterator {
	return NewBsonIterator(bson)
}

func (bson *Bson) Length() int {
	if !bson.finished {
		panic("the bson is unfinished")
	}

	return int(bytesToInt32(bson.raw[bson.offset:]))
}

func (bson *Bson) String() string {
	var err error
	buf := bytes.NewBufferString("{")
	it := bson.Iterator()
	for it.Next() {
		switch it.BsonType() {
		case BsonTypeFloat64:
			_, err = fmt.Fprintf(buf, `"%s":%v`, it.Name(), it.Float64())
		case BsonTypeString:
			_, err = fmt.Fprintf(buf, `"%s":"%s"`, it.Name(), it.UTF8String())
		case BsonTypeBson:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.Bson().String())
		case BsonTypeArray:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.BsonArray().String())
		case BsonTypeBinary:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.Binary().String())
		case BsonTypeObjectId:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.ObjectId().String())
		case BsonTypeBool:
			_, err = fmt.Fprintf(buf, `"%s":%v`, it.Name(), it.Bool())
		case BsonTypeDate:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.Date().String())
		case BsonTypeNull:
			_, err = fmt.Fprintf(buf, `"%s":null`, it.Name())
		case BsonTypeRegEx:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.RegEx().String())
		case BsonTypeInt32:
			_, err = fmt.Fprintf(buf, `"%s":%v`, it.Name(), it.Int32())
		case BsonTypeTimestamp:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), it.Timestamp().String())
		case BsonTypeInt64:
			_, err = fmt.Fprintf(buf, `"%s":%v`, it.Name(), it.Int64())
		case BsonTypeMaxKey:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), MaxKey.String())
		case BsonTypeMinKey:
			_, err = fmt.Fprintf(buf, `"%s":%s`, it.Name(), MinKey.String())
		case BsonTypeEOD:
			// END
		case BsonTypeUndefined: // deprecated
			fallthrough
		case BsonTypeDBPointer: // deprecated
			fallthrough
		case BsonTypeCode: // not support
			fallthrough
		case BsonTypeSymbol: // deprecated
			fallthrough
		case BsonTypeCodeWScope: // not support
			fallthrough
		default:
			panic(fmt.Errorf("invalid bson type: %v", it.BsonType()))
		}

		if err != nil {
			panic(fmt.Sprintf("failed to convert bson to string: %v", err))
		}

		if it.More() {
			_, err = buf.WriteString(", ")
			if err != nil {
				panic(fmt.Sprintf("failed to convert bson to string: %v", err))
			}
		}
	}

	_, err = buf.WriteString("}")
	if err != nil {
		panic(fmt.Sprintf("failed to convert bson to string: %v", err))
	}

	return buf.String()
}

func (bson *Bson) Map() Map {
	if !bson.finished {
		panic("the bson is unfinished")
	}

	m := make(Map)

	it := bson.Iterator()
	for it.Next() {
		switch it.BsonType() {
		case BsonTypeBson:
			m[it.Name()] = it.Bson().Map()
		case BsonTypeArray:
			m[it.Name()] = it.BsonArray().MapSlice()
		default:
			m[it.Name()] = it.Value()
		}
	}

	return m
}

func (bson *Bson) Doc() Doc {
	if !bson.finished {
		panic("the bson is unfinished")
	}

	d := []DocElement{}

	it := bson.Iterator()
	for it.Next() {
		var val interface{}
		switch it.BsonType() {
		case BsonTypeBson:
			val = it.Bson().Doc()
		case BsonTypeArray:
			val = it.BsonArray().DocSlice()
		default:
			val = it.Value()
		}
		d = append(d, DocElement{Name: it.Name(), Value: val})
	}

	return d
}

func (bson *Bson) Struct(s interface{}) {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic("s must be struct pointer")
	}
	docToStruct(v.Elem(), bson.Doc())
}
