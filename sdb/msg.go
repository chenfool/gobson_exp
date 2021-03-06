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

package sdb

import (
	"encoding/binary"

	"io"

	"fmt"

	"errors"

	"github.com/davidli2010/gobson_exp/bson"
)

type Msg interface {
	Size() int32
	Encode(io.Writer, binary.ByteOrder) error
	Decode(io.Reader, binary.ByteOrder) error
}

// SysInfoMsg---------------------------------------

const (
	sysInfoSpecial          = 0xFFFFFFFF
	sysInfoEyeCatcher       = 0xFFFEFDFC
	sysInfoEyeCatcherRevert = 0xFCFDFEFF
)

type MsgCode uint32

const (
	RspMsgMask   = MsgCode(0x80000000)
	RspMsgUnmask = MsgCode(0x7FFFFFFF)

	UpdateReqMsg = MsgCode(2001)
	UpdateRspMsg = UpdateReqMsg | RspMsgMask

	InsertReqMsg = MsgCode(2002)
	InsertRspMsg = InsertReqMsg | RspMsgMask

	QueryReqMsg = MsgCode(2004)
	QueryRspMsg = QueryReqMsg | RspMsgMask

	DeleteReqMsg = MsgCode(2006)
	DeleteRspMsg = DeleteReqMsg | RspMsgMask

	DisconnectReqMsg = MsgCode(2008)
)

type SysInfoMsgHeader struct {
	Special    uint32
	EyeCatcher uint32
	Length     int32
}

const sysInfoMsgHeaderSize = 12

func (m *SysInfoMsgHeader) Size() int32 {
	return sysInfoMsgHeaderSize
}

func (m *SysInfoMsgHeader) Encode(w io.Writer, order binary.ByteOrder) error {
	var b [12]byte
	buf := b[:]
	order.PutUint32(buf, m.Special)
	order.PutUint32(buf[4:], m.EyeCatcher)
	order.PutUint32(buf[8:], uint32(m.Length))
	_, err := w.Write(buf)
	return err
}

func (m *SysInfoMsgHeader) Decode(r io.Reader, order binary.ByteOrder) error {
	var b [12]byte
	buf := b[:]
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	m.Special = order.Uint32(buf)
	m.EyeCatcher = order.Uint32(buf[4:])
	m.Length = int32(order.Uint32(buf[8:]))
	return nil
}

type SysInfoRequest struct {
	SysInfoMsgHeader
}

const sysInfoRequestSize = sysInfoMsgHeaderSize

func NewSysInfoRequest() *SysInfoRequest {
	return &SysInfoRequest{
		SysInfoMsgHeader{
			Special:    sysInfoSpecial,
			EyeCatcher: sysInfoEyeCatcher,
			Length:     sysInfoRequestSize,
		},
	}
}

func (m *SysInfoRequest) Size() int32 {
	return sysInfoRequestSize
}

func (m *SysInfoRequest) Encode(w io.Writer, order binary.ByteOrder) error {
	return m.SysInfoMsgHeader.Encode(w, order)
}

func (m *SysInfoRequest) Decode(r io.Reader, order binary.ByteOrder) error {
	return m.SysInfoMsgHeader.Decode(r, order)
}

var sysInfoRequest = SysInfoRequest{
	SysInfoMsgHeader{
		Special:    sysInfoSpecial,
		EyeCatcher: sysInfoEyeCatcher,
		Length:     sysInfoRequestSize,
	},
}

type SysInfoReply struct {
	SysInfoMsgHeader
	OSType int32
}

const sysInfoReplySize = 128

func (m *SysInfoReply) Size() int32 {
	return sysInfoReplySize
}

func (m *SysInfoReply) Encode(w io.Writer, order binary.ByteOrder) error {
	if err := m.SysInfoMsgHeader.Encode(w, order); err != nil {
		return err
	}
	var b [sysInfoReplySize - sysInfoMsgHeaderSize]byte
	buf := b[:]
	order.PutUint32(buf, uint32(m.OSType))
	_, err := w.Write(buf)
	return err
}

func (m *SysInfoReply) Decode(r io.Reader, order binary.ByteOrder) error {
	if err := m.SysInfoMsgHeader.Decode(r, order); err != nil {
		return err
	}
	if m.Length != sysInfoReplySize {
		return errors.New("invalid sysinfo reply size")
	}
	var b [sysInfoReplySize - sysInfoMsgHeaderSize]byte
	buf := b[:]
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	m.OSType = int32(order.Uint32(buf))
	return nil
}

// MsgHeader-----------------------------

type MsgHeader struct {
	Length    int32
	OpCode    MsgCode
	Tid       uint32
	RouteId   uint64
	RequestId uint64
}

const msgHeaderSize = 28

func (m *MsgHeader) Size() int32 {
	return msgHeaderSize
}

func (m *MsgHeader) Encode(w io.Writer, order binary.ByteOrder) error {
	var b [msgHeaderSize]byte
	buf := b[:]
	order.PutUint32(buf, uint32(m.Length))
	order.PutUint32(buf[4:], uint32(m.OpCode))
	order.PutUint32(buf[8:], m.Tid)
	order.PutUint64(buf[12:], m.RouteId)
	order.PutUint64(buf[20:], m.RequestId)
	_, err := w.Write(buf)
	return err
}

func (m *MsgHeader) Decode(r io.Reader, order binary.ByteOrder) error {
	var b [msgHeaderSize]byte
	buf := b[:]
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	m.Length = int32(order.Uint32(buf))
	m.OpCode = MsgCode(order.Uint32(buf[4:]))
	m.Tid = order.Uint32(buf[8:])
	m.RouteId = order.Uint64(buf[12:])
	m.RequestId = order.Uint64(buf[20:])
	return nil
}

// ReplyMsg------------------------------

type ReplyMsg struct {
	MsgHeader
	ContextId int64
	Flags     int32
	StartFrom int32
	ReturnNum int32
	Error     string
}

func (m *ReplyMsg) Size() int32 {
	return m.MsgHeader.Size() + 20
}

func (m *ReplyMsg) Decode(r io.Reader, order binary.ByteOrder) error {
	if err := m.MsgHeader.Decode(r, order); err != nil {
		return err
	}

	if m.Length < m.Size() {
		return fmt.Errorf("invalid msg length: expect %d, actual %d", m.Size(), m.Length)
	}

	var b [20]byte
	buf := b[:]
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}

	m.ContextId = int64(order.Uint64(buf))
	m.Flags = int32(order.Uint32(buf[8:]))
	m.StartFrom = int32(order.Uint32(buf[12:]))
	m.ReturnNum = int32(order.Uint32(buf[16:]))

	if m.Flags == 0 {
		return nil
	}

	buf = make([]byte, m.Length-m.Size())
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	errInfo := bson.NewBson(buf)
	m.Error = errInfo.String()

	return nil
}

// AuthMsg-------------------------------

type AuthMsg struct {
	MsgHeader
	Data bson.Bson
}

// DisconnectMsg-------------------------

type DisconnectMsg struct {
	MsgHeader
}

func NewDisconnectMsg() *DisconnectMsg {
	return &DisconnectMsg{
		MsgHeader{
			Length: msgHeaderSize,
			OpCode: DisconnectReqMsg,
		},
	}
}

func (m *DisconnectMsg) Encode(w io.Writer, order binary.ByteOrder) error {
	return m.MsgHeader.Encode(w, order)
}

// QueryMsg------------------------------

type QueryMsg struct {
	MsgHeader
	Version    int32
	W          int16
	padding    uint16
	Flags      int32
	NameLength int32
	SkipNum    int64
	ReturnNum  int64
	Name       []byte
	Where      *bson.Bson
	Select     *bson.Bson
	OrderBy    *bson.Bson
	Hint       *bson.Bson
}

func (m *QueryMsg) FixedSize() int32 {
	return m.MsgHeader.Size() + 32
}

func (m *QueryMsg) Encode(w io.Writer, order binary.ByteOrder) error {
	if err := m.MsgHeader.Encode(w, order); err != nil {
		return err
	}

	var b [32]byte
	buf := b[:]
	order.PutUint32(buf, uint32(m.Version))
	order.PutUint16(buf[4:], uint16(m.W))
	order.PutUint16(buf[6:], m.padding)
	order.PutUint32(buf[8:], uint32(m.Flags))
	order.PutUint32(buf[12:], uint32(m.NameLength))
	order.PutUint64(buf[16:], uint64(m.SkipNum))
	order.PutUint64(buf[24:], uint64(m.ReturnNum))
	if _, err := w.Write(buf); err != nil {
		return err
	}

	if _, err := w.Write(m.Name); err != nil {
		return err
	}

	paddingLen := alignedSize(m.NameLength+1, 4) - m.NameLength
	if paddingLen > 0 {
		if _, err := w.Write(make([]byte, paddingLen)); err != nil {
			return err
		}
	}

	if m.Where != nil {
		if err := writeBson(w, *m.Where); err != nil {
			return err
		}
	}

	if m.Select != nil {
		if err := writeBson(w, *m.Select); err != nil {
			return err
		}
	}

	if m.OrderBy != nil {
		if err := writeBson(w, *m.OrderBy); err != nil {
			return err
		}
	}

	if m.Hint != nil {
		if err := writeBson(w, *m.Hint); err != nil {
			return err
		}
	}

	return nil
}

func writeBson(w io.Writer, b bson.Bson) error {
	if _, err := w.Write(b.Raw()); err != nil {
		return err
	}

	paddingLen := alignedSize(int32(b.Length()), 4) - int32(b.Length())
	if paddingLen > 0 {
		if _, err := w.Write(make([]byte, paddingLen)); err != nil {
			return err
		}
	}

	return nil
}

// InsertMsg------------------------------

type InsertMsg struct {
	MsgHeader
	Version    int32
	W          int16
	padding    uint16
	Flags      int32
	NameLength int32
	Name       []byte
	Doc        *bson.Bson
}

func (m *InsertMsg) FixedSize() int32 {
	return m.MsgHeader.Size() + 16
}

func (m *InsertMsg) Encode(w io.Writer, order binary.ByteOrder) error {
	if err := m.MsgHeader.Encode(w, order); err != nil {
		return err
	}

	var b [16]byte
	buf := b[:]
	order.PutUint32(buf, uint32(m.Version))
	order.PutUint16(buf[4:], uint16(m.W))
	order.PutUint16(buf[6:], m.padding)
	order.PutUint32(buf[8:], uint32(m.Flags))
	order.PutUint32(buf[12:], uint32(m.NameLength))
	if _, err := w.Write(buf); err != nil {
		return err
	}

	if _, err := w.Write(m.Name); err != nil {
		return err
	}

	paddingLen := alignedSize(m.NameLength+1, 4) - m.NameLength
	if paddingLen > 0 {
		if _, err := w.Write(make([]byte, paddingLen)); err != nil {
			return err
		}
	}

	if m.Doc != nil {
		if err := writeBson(w, *m.Doc); err != nil {
			return err
		}
	}

	return nil
}

// DeleteMsg------------------------------

type DeleteMsg struct {
	MsgHeader
	Version    int32
	W          int16
	padding    uint16
	Flags      int32
	NameLength int32
	Name       []byte
	Condition  *bson.Bson
	Hint       *bson.Bson
}

func (m *DeleteMsg) FixedSize() int32 {
	return m.MsgHeader.Size() + 16
}

func (m *DeleteMsg) Encode(w io.Writer, order binary.ByteOrder) error {
	if err := m.MsgHeader.Encode(w, order); err != nil {
		return err
	}

	var b [16]byte
	buf := b[:]
	order.PutUint32(buf, uint32(m.Version))
	order.PutUint16(buf[4:], uint16(m.W))
	order.PutUint16(buf[6:], m.padding)
	order.PutUint32(buf[8:], uint32(m.Flags))
	order.PutUint32(buf[12:], uint32(m.NameLength))
	if _, err := w.Write(buf); err != nil {
		return err
	}

	if _, err := w.Write(m.Name); err != nil {
		return err
	}

	paddingLen := alignedSize(m.NameLength+1, 4) - m.NameLength
	if paddingLen > 0 {
		if _, err := w.Write(make([]byte, paddingLen)); err != nil {
			return err
		}
	}

	cond := m.Condition
	if cond == nil {
		cond = emptyBson
	}

	if err := writeBson(w, *cond); err != nil {
		return err
	}

	hint := m.Hint
	if hint == nil {
		hint = emptyBson
	}

	if err := writeBson(w, *hint); err != nil {
		return err
	}

	return nil
}

// UpdateMsg------------------------------

type UpdateMsg struct {
	MsgHeader
	Version    int32
	W          int16
	padding    uint16
	Flags      int32
	NameLength int32
	Name       []byte
	Condition  *bson.Bson
	Rule       *bson.Bson
	Hint       *bson.Bson
}

func (m *UpdateMsg) FixedSize() int32 {
	return m.MsgHeader.Size() + 16
}

func (m *UpdateMsg) Encode(w io.Writer, order binary.ByteOrder) error {
	if err := m.MsgHeader.Encode(w, order); err != nil {
		return err
	}

	var b [16]byte
	buf := b[:]
	order.PutUint32(buf, uint32(m.Version))
	order.PutUint16(buf[4:], uint16(m.W))
	order.PutUint16(buf[6:], m.padding)
	order.PutUint32(buf[8:], uint32(m.Flags))
	order.PutUint32(buf[12:], uint32(m.NameLength))
	if _, err := w.Write(buf); err != nil {
		return err
	}

	if _, err := w.Write(m.Name); err != nil {
		return err
	}

	paddingLen := alignedSize(m.NameLength+1, 4) - m.NameLength
	if paddingLen > 0 {
		if _, err := w.Write(make([]byte, paddingLen)); err != nil {
			return err
		}
	}

	cond := m.Condition
	if cond == nil {
		cond = emptyBson
	}
	if err := writeBson(w, *cond); err != nil {
		return err
	}

	rule := m.Rule
	if rule == nil {
		rule = emptyBson
	}
	if err := writeBson(w, *rule); err != nil {
		return err
	}

	hint := m.Hint
	if hint == nil {
		hint = emptyBson
	}
	if err := writeBson(w, *hint); err != nil {
		return err
	}

	return nil
}
