package logger

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func newEncoder() *goModEncoder {
	return &goModEncoder{
		Encoder:       zapcore.NewConsoleEncoder(fieldEncoderConfig),
		EncoderConfig: externalEncoderConfig,
	}
}

type goModEncoder struct {
	zapcore.Encoder
	zapcore.EncoderConfig

	indent int
}

func (c goModEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := pool.Get()

	arr := &sliceArrayEncoder{}
	c.EncodeTime(ent.Time, arr)
	c.EncodeLevel(ent.Level, arr)
	c.EncodeName(ent.LoggerName, arr)
	if c.indent > 0 {
		arr.AppendString(strings.Repeat("-", c.indent))
	}
	if ent.Caller.Defined && c.EncodeCaller != nil {
		c.EncodeCaller(ent.Caller, arr)
	}
	arr.AppendString(ent.Message)

	for i := range arr.elems {
		if i > 0 {
			line.AppendByte(' ')
		}
		fmt.Fprint(line, arr.elems[i])
	}

	b, err := c.Encoder.EncodeEntry(zapcore.Entry{}, fields)
	switch {
	case err != nil:
		return nil, err
	case b.Len() > 0:
		line.AppendByte(' ')
		if _, err = line.Write(b.Bytes()); err != nil {
			return nil, err
		}
	default:
		line.AppendString("\n")
	}

	return line, nil
}

var (
	pool = buffer.NewPool()

	externalEncoderConfig = zapcore.EncoderConfig{
		LevelKey:       "level",
		TimeKey:        "time",
		MessageKey:     "msg",
		CallerKey:      "caller",
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
		EncodeTime:     func(t time.Time, enc zapcore.PrimitiveArrayEncoder) { enc.AppendString(t.Format("15:04:05")) },
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	fieldEncoderConfig = zapcore.EncoderConfig{
		// We omit any of the keys so that we don't print any of the already previously printed fields.
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
		EncodeTime:     func(t time.Time, enc zapcore.PrimitiveArrayEncoder) { enc.AppendString(t.Format("15:04:05")) },
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
	}
)

type sliceArrayEncoder struct {
	elems []interface{}
}

func (s *sliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	enc := &sliceArrayEncoder{}
	err := v.MarshalLogArray(enc)
	s.elems = append(s.elems, enc.elems)
	return err
}

func (s *sliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	err := v.MarshalLogObject(m)
	s.elems = append(s.elems, m.Fields)
	return err
}

func (s *sliceArrayEncoder) AppendReflected(v interface{}) error {
	s.elems = append(s.elems, v)
	return nil
}

func (s *sliceArrayEncoder) AppendBool(v bool)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendByteString(v []byte)      { s.elems = append(s.elems, string(v)) }
func (s *sliceArrayEncoder) AppendComplex128(v complex128)  { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendComplex64(v complex64)    { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendFloat64(v float64)        { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendFloat32(v float32)        { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt(v int)                { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt64(v int64)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt32(v int32)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt16(v int16)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt8(v int8)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendString(v string)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendTime(v time.Time)         { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint(v uint)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint64(v uint64)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint32(v uint32)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint16(v uint16)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint8(v uint8)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUintptr(v uintptr)        { s.elems = append(s.elems, v) }
