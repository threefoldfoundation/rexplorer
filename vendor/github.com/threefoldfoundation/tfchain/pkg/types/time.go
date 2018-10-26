package types

import (
	"encoding/json"
	"io"
	"time"

	"github.com/rivine/rivine/types"
	"github.com/threefoldfoundation/tfchain/pkg/encoding"
)

// CompactTimestamp binary marshals the regular Unix Epoch (seconds) Timestamp,
// in a custom format, such that it only requires 3 bytes in space.
// It does so by being only accurate up to 60 seconds, and by starting the Timestamp
// since `CompactTimestampNullpoint`.
type CompactTimestamp uint64

const (
	// CompactTimestampNullpoint defines the time at which the Timestamp starts (~Jan '18)
	CompactTimestampNullpoint CompactTimestamp = 1515000000
	// CompactTimestampAccuracyInSeconds defines the lowest possible value that gets recorded in seconds
	CompactTimestampAccuracyInSeconds CompactTimestamp = 60
)

// SiaTimestampAsCompactTimestamp converts a Sia/Rivine Timestamp to a Tfchain Compact timestamp.
func SiaTimestampAsCompactTimestamp(ts types.Timestamp) CompactTimestamp {
	ct := CompactTimestamp(ts)
	ct -= ct % CompactTimestampAccuracyInSeconds
	return ct
}

// NowAsCompactTimestamp returns the current Epoch Unix seconds time as a Tfchain Compact timestamp.
func NowAsCompactTimestamp() CompactTimestamp {
	return CompactTimestamp(time.Now().Unix())
}

// MarshalSia implements SiaMarshaler.MarshalSia
func (cts CompactTimestamp) MarshalSia(w io.Writer) error {
	if cts < CompactTimestampNullpoint {
		return encoding.MarshalUint24(w, 0)
	}
	return encoding.MarshalUint24(w, cts.UInt32())
}

// UnmarshalSia implements SiaUnmarshaler.UnmarshalSia
func (cts *CompactTimestamp) UnmarshalSia(r io.Reader) error {
	x, err := encoding.UnmarshalUint24(r)
	if err != nil {
		return err
	}
	cts.SetUInt32(x)
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.UnmarshalJSON
func (cts *CompactTimestamp) UnmarshalJSON(b []byte) error {
	var x uint32
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}
	*cts = CompactTimestamp(x)
	*cts -= *cts % CompactTimestampAccuracyInSeconds
	return nil
}

// SiaTimestamp returns this CompactTimestamp as a Unix Epoch Seconds timestamp,
// the type wrapped by a Sia/Rivine timestamp.
func (cts CompactTimestamp) SiaTimestamp() types.Timestamp {
	return types.Timestamp(cts)
}

// UInt32 returns this CompactTimestamp as an uint32 number.
func (cts CompactTimestamp) UInt32() uint32 {
	return uint32((cts - CompactTimestampNullpoint) / CompactTimestampAccuracyInSeconds)
}

// SetUInt32 sets an uint32 version of this CompactTimestamp as the internal value of this compact time stmap.
func (cts *CompactTimestamp) SetUInt32(x uint32) {
	*cts = (CompactTimestamp(x) * CompactTimestampAccuracyInSeconds) + CompactTimestampNullpoint
}
