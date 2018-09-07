package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/glycerine/greenpack/msgp"
	"github.com/gogo/protobuf/proto"
)

// Type identifies a type of Encoder
type Type uint8

// the Type enumeration values
const (
	TypeMessagePack Type = iota
	TypeProtocolBuffer
	TypeJSON
)

// string versions of the Type enumeration values
const (
	TypeMessagePackStr    = "msgp"
	TypeProtocolBufferStr = "protobuf"
	TypeJSONStr           = "json"
)

// String implements flag.Value.String and fmt.Stringer.String
func (et Type) String() string {
	switch et {
	case TypeMessagePack:
		return TypeMessagePackStr
	case TypeProtocolBuffer:
		return TypeProtocolBufferStr
	case TypeJSON:
		return TypeJSONStr
	default:
		panic("unknown Typee " + strconv.Itoa(int(et)))
	}
}

// Set implements flag.Value.Set
func (et *Type) Set(str string) error {
	switch str {
	case TypeMessagePackStr:
		*et = TypeMessagePack
	case TypeProtocolBufferStr:
		*et = TypeProtocolBuffer
	case TypeJSONStr:
		*et = TypeJSON
	default:
		return fmt.Errorf("unknown Type string: %s", str)
	}
	return nil
}

// Type implements pflag.Value.Type
func (et Type) Type() string {
	return "EncodingType"
}

// LoadString implements StringLoader.LoadString,
// piggy-backing on the flag.Value.Set implementation of this type.
func (et *Type) LoadString(str string) error {
	return et.Set(str)
}

// Encoder defines a universal interface that
// all EncodingType implementations have to adhere.
type Encoder interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
}

// NewEncoder creates an Encoder, using the standard Encoder implementation
// matched for the given Encoding Type.
func NewEncoder(et Type) (Encoder, error) {
	switch et {
	case TypeMessagePack:
		return NewMessagePackEncoder(), nil
	case TypeProtocolBuffer:
		return NewProtocolBufferEncoder(), nil
	case TypeJSON:
		return NewJSONEncoder(), nil
	default:
		return nil, fmt.Errorf("cannot create encoder for unknown Typee %d", et)
	}
}

type (
	// MessagePackEncoder defines the standard implementation for the
	// MessagePack Encoding Type, using a single bytes.Buffer for all marshal calls,
	// and using a single shared instance of the Reader/Writer types of the github.com/tinylib/msgp/msgp pkg internally.
	MessagePackEncoder struct {
		w  *msgp.Writer
		wb *bytes.Buffer
		r  *msgp.Reader
		br *bytes.Reader
	}
	// ProtocolBufferEncoder defines the standard implementation for the
	// ProtocolBuffer Encoding Type, using a single proto.Buffer for all marshal calls,
	// of the github.com/gogo/protobuf/proto internally.
	ProtocolBufferEncoder struct {
		buffer *proto.Buffer
	}
	// JSONEncoder defines the standard implementation for the
	// JSON Encoding Type, using a single bytes.Buffer for all marshal calls,
	// and using the Encoder type of the std encoding/json pkg internally.
	JSONEncoder struct {
		e  *json.Encoder
		wb *bytes.Buffer
	}
)

// custom ProtocolBuffer interfaces,
// as to allow our high level structures to implement these,
// such that they can automatically play nice with ProtocolBuffer as well,
// without having to change any of the non-encoding logic.
type (
	// ProtocolBufferWriter is used by ProtocolBufferMarshaler implementations
	// to marshal (=write) themselves as a ProtocolBuffer Message.
	ProtocolBufferWriter interface {
		Marshal(pb proto.Message) error
	}
	// ProtocolBufferReader is used by ProtocolBufferUnmarshaler implementations
	// to unmarshal (=read) their encoded form as a given ProtocolBuffer Message
	ProtocolBufferReader interface {
		Unmarshal(pb proto.Message) error
	}
	// ProtocolBufferMarshaler is the interface implemented
	// by types that know how to write themselves
	// as ProtocolBuffer Messages into a given ProtocolBufferWriter.
	ProtocolBufferMarshaler interface {
		ProtocolBufferMarshal(ProtocolBufferWriter) error
	}
	// ProtocolBufferUnmarshaler is the interface implemented
	// by types that know how to read themselves
	// as ProtocolBuffer Messages from a given ProtocolBufferReader.
	ProtocolBufferUnmarshaler interface {
		ProtocolBufferUnmarshal(ProtocolBufferReader) error
	}
)

// NewMessagePackEncoder creates a new MessagePackEncoder,
// allocating a new bytes buffer and a MessagePack Writer (encapsulating the buffer).
//
// See MessagePackEncoder for more information.
func NewMessagePackEncoder() *MessagePackEncoder {
	wb := bytes.NewBuffer(nil)
	br := bytes.NewReader(nil)
	return &MessagePackEncoder{
		w:  msgp.NewWriter(wb),
		wb: wb,
		r:  msgp.NewReader(br),
		br: br,
	}
}

// Marshal implements Encoder.Marshal
func (encoder MessagePackEncoder) Marshal(v interface{}) ([]byte, error) {
	switch tv := v.(type) {
	case msgp.Encodable:
		encoder.wb.Reset()
		encoder.w.Reset(encoder.wb)

		err := tv.EncodeMsg(encoder.w)
		if err != nil {
			return nil, fmt.Errorf("failed to message pack Encodable value: %v", err)
		}
		err = encoder.w.Flush()
		if err != nil {
			return nil, fmt.Errorf("failed to flush message packed Encodable value: %v", err)
		}

		return encoder.wb.Bytes(), nil

	default:
		return nil, fmt.Errorf("cannot message pack unexpected value %[1]v (%[1]T)", v)
	}
}

// Unmarshal implements encoding.Unmarshal
func (encoder MessagePackEncoder) Unmarshal(data []byte, v interface{}) error {
	switch tv := v.(type) {
	case msgp.Decodable:
		encoder.br.Reset(data)
		encoder.r.Reset(encoder.br)
		err := tv.DecodeMsg(encoder.r)
		if err != nil {
			return fmt.Errorf("failed to message unpack Decodable value: %v", err)
		}
		return nil

	default:
		return fmt.Errorf("cannot message unpack unexpected value %[1]v (%[1]T)", v)
	}
}

// NewJSONEncoder creates a new JSONEncoder,
// allocating a new bytes buffer and a (std) JSON Encoder (encapsulating the buffer).
//
// See JSONEncoder for more information.
func NewJSONEncoder() *JSONEncoder {
	wb := bytes.NewBuffer(nil)
	return &JSONEncoder{
		e:  json.NewEncoder(wb),
		wb: wb,
	}
}

// Marshal implements Encoder.Marshal
func (encoder JSONEncoder) Marshal(v interface{}) ([]byte, error) {
	encoder.wb.Reset()
	err := encoder.e.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value to shared buffer: %v", err)
	}
	return encoder.wb.Bytes(), nil
}

// Unmarshal implements encoder.Unmarshal
func (encoder JSONEncoder) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// NewProtocolBufferEncoder creates a new ProtocolBufferEncoder,
// allocating a new proto buffer used for marshaling.
//
// See ProtocolBufferEncoder for more information.
func NewProtocolBufferEncoder() *ProtocolBufferEncoder {
	return &ProtocolBufferEncoder{
		buffer: proto.NewBuffer(nil),
	}
}

// Marshal implements Encoder.Marshal
func (encoder *ProtocolBufferEncoder) Marshal(v interface{}) ([]byte, error) {
	switch tv := v.(type) {
	case ProtocolBufferMarshaler:
		encoder.buffer.Reset()
		err := tv.ProtocolBufferMarshal(encoder.buffer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value as protocol buffer message: %v", err)
		}
		return encoder.buffer.Bytes(), nil

	default:
		return nil, fmt.Errorf("cannot marshal unexpected value %[1]v (%[1]T) "+
			"as a protocol buffer message", v)
	}
}

// Unmarshal implements encoder.Unmarshal
func (encoder *ProtocolBufferEncoder) Unmarshal(data []byte, v interface{}) error {
	switch tv := v.(type) {
	case ProtocolBufferUnmarshaler:
		err := tv.ProtocolBufferUnmarshal(proto.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to unmarshal protocol buffer message: %v", err)
		}
		return nil

	default:
		return fmt.Errorf("cannot unmarshal protocol buffer message as value %[1]v (%[1]T)", v)
	}
}
