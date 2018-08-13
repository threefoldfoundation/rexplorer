package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tinylib/msgp/msgp"
)

// Type identifies a type of Encoder
type Type uint8

// the Type enumeration values
const (
	TypeMessagePack Type = iota
	TypeJSON
)

// string versions of the Type enumeration values
const (
	TypeMessagePackStr = "msgp"
	TypeJSONStr        = "json"
)

// String implements flag.Value.String and fmt.Stringer.String
func (et Type) String() string {
	switch et {
	case TypeMessagePack:
		return TypeMessagePackStr
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
	case TypeJSONStr:
		*et = TypeJSON
	default:
		return fmt.Errorf("unknown Typee string: %s", str)
	}
	return nil
}

// Type implements pflag.Value.Type
func (et Type) Type() string {
	return "EncdodingType"
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
	case TypeJSON:
		return NewJSONEncoder(), nil
	default:
		return nil, fmt.Errorf("cannot create encoder for unknown Typee %d", et)
	}
}

type (
	// MessagePackEncoder defines the standard implementation for the
	// MessagePack Encoding Type, using a single bytes.Buffer for all marshal calls,
	// and using the Writer type of the github.com/tinylib/msgp/msgp pkg internally.
	MessagePackEncoder struct {
		w  *msgp.Writer
		wb *bytes.Buffer
	}
	// JSONEncoder defines the standard implementation for the
	// JSON Encoding Type, using a single bytes.Buffer for all marshal calls,
	// and using the Encoder type of the std encoding/json pkg internally.
	JSONEncoder struct {
		e  *json.Encoder
		wb *bytes.Buffer
	}
)

// NewMessagePackEncoder creates a new MessagePackEncoder,
// allocating a new bytes buffer and a MessagePack Writer (encapsulating the buffer).
//
// See MessagePackEncoder for more information.
func NewMessagePackEncoder() *MessagePackEncoder {
	wb := bytes.NewBuffer(nil)
	return &MessagePackEncoder{
		w:  msgp.NewWriter(wb),
		wb: wb,
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
		r := msgp.NewReader(bytes.NewReader(data))
		err := tv.DecodeMsg(r)
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
