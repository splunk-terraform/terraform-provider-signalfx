package messages

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"math"

	"github.com/signalfx/signalfx-go/idtool"
)

type BinaryPayload struct {
	Type ValType
	TSID idtool.ID
	Val  [8]byte
}

// Value returns the numeric value as an interface{}.
func (bp *BinaryPayload) Value() interface{} {
	switch bp.Type {
	case ValTypeLong:
		return bp.Int64()
	case ValTypeDouble:
		return bp.Float64()
	case ValTypeInt:
		return bp.Int32()
	default:
		return nil
	}
}

func (bp *BinaryPayload) Int64() int64 {
	n := binary.BigEndian.Uint64(bp.Val[:])
	return int64(n)
}

func (bp *BinaryPayload) Float64() float64 {
	bits := binary.BigEndian.Uint64(bp.Val[:])
	return math.Float64frombits(bits)
}

func (bp *BinaryPayload) Int32() int32 {
	var n int32
	_ = binary.Read(bytes.NewBuffer(bp.Val[:]), binary.BigEndian, &n)
	return n
}

// DataMessage is a set of datapoints that share a common timestamp
type DataMessage struct {
	BaseMessage
	BaseChannelMessage
	TimestampedMessage
	Payloads []BinaryPayload
}

func (dm *DataMessage) String() string {
	pls := make([]map[string]interface{}, 0)
	for _, pl := range dm.Payloads {
		pls = append(pls, map[string]interface{}{
			"type":  pl.Type,
			"tsid":  pl.TSID,
			"value": pl.Value(),
		})
	}

	return fmt.Sprintf("%v", map[string]interface{}{
		"channel":   dm.Channel(),
		"timestamp": dm.Timestamp(),
		"payloads":  pls,
	})
}

type binaryMessageHeader struct {
	TimestampMillis uint64
	ElementCount    uint32
}

type ValType uint8

const (
	ValTypeLong   ValType = 1
	ValTypeDouble ValType = 2
	ValTypeInt    ValType = 3
)

func (vt ValType) String() string {
	switch vt {
	case ValTypeLong:
		return "long"
	case ValTypeDouble:
		return "double"
	case ValTypeInt:
		return "int32"
	}
	return "Unknown"
}

// The first 20 bytes of every binary websocket message from the backend.
// https://developers.signalfx.com/signalflow_analytics/rest_api_messages/stream_messages_specification.html#_binary_encoding_of_websocket_messages
type msgHeader struct {
	Version     uint8
	MessageType uint8
	Flags       uint8
	Reserved    uint8
	Channel     [16]byte
}

const (
	compressed  uint8 = 1 << iota
	jsonEncoded       = 1 << iota
)

func parseBinaryHeader(msg []byte) (string, bool /* isCompressed */, bool /* isJSON */, []byte /* rest of message */, error) {
	if len(msg) <= 20 {
		return "", false, false, nil, fmt.Errorf("expected SignalFlow message of at least 21 bytes, got %d bytes", len(msg))
	}

	r := bytes.NewReader(msg[:20])
	var header msgHeader
	err := binary.Read(r, binary.BigEndian, &header)
	if err != nil {
		return "", false, false, nil, err
	}

	isCompressed := header.Flags&compressed != 0
	isJSON := header.Flags&jsonEncoded != 0

	return string(header.Channel[:bytes.IndexByte(header.Channel[:], 0)]), isCompressed, isJSON, msg[20:], err
}

func parseBinaryMessage(msg []byte) (Message, error) {
	channel, isCompressed, isJSON, rest, err := parseBinaryHeader(msg)
	if err != nil {
		return nil, err
	}

	if isCompressed {
		reader, err := gzip.NewReader(bytes.NewReader(rest))
		if err != nil {
			return nil, err
		}
		rest, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	}

	if isJSON {
		return nil, errors.New("cannot handle json binary message")
	}

	r := bytes.NewReader(rest[:12])
	var header binaryMessageHeader
	err = binary.Read(r, binary.BigEndian, &header)
	if err != nil {
		return nil, err
	}

	var payloads []BinaryPayload
	for i := 0; i < int(header.ElementCount); i++ {
		r := bytes.NewReader(rest[12+17*i : 12+17*(i+1)])
		var payload BinaryPayload
		if err := binary.Read(r, binary.BigEndian, &payload); err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}

	return &DataMessage{
		BaseMessage: BaseMessage{
			Typ: DataType,
		},
		BaseChannelMessage: BaseChannelMessage{
			Chan: channel,
		},
		TimestampedMessage: TimestampedMessage{
			TimestampMillis: header.TimestampMillis,
		},
		Payloads: payloads,
	}, nil
}
