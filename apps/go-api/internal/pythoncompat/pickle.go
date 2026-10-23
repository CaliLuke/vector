package pythoncompat

import (
	"encoding/binary"
	"fmt"
	"math"
)

// EncodeFloatListPickle writes the protocol-4 pickle form used for Django's
// cached throttle histories.
func EncodeFloatListPickle(values []float64) []byte {
	payload := []byte{']', 0x94}
	if len(values) == 1 {
		payload = append(payload, 'G')
		payload = binary.BigEndian.AppendUint64(payload, math.Float64bits(values[0]))
		payload = append(payload, 'a')
	} else if len(values) > 1 {
		payload = append(payload, '(')
		for _, value := range values {
			payload = append(payload, 'G')
			payload = binary.BigEndian.AppendUint64(payload, math.Float64bits(value))
		}
		payload = append(payload, 'e')
	}
	payload = append(payload, '.')
	encoded := []byte{0x80, 0x04, 0x95}
	encoded = binary.LittleEndian.AppendUint64(encoded, uint64(len(payload)))
	return append(encoded, payload...)
}

// DecodeFloatListPickle reads the protocol-4 float-list subset Django uses
// for cached throttle histories.
func DecodeFloatListPickle(data []byte) ([]float64, error) {
	values := make([]float64, 0, 60)
	for index := 0; index < len(data); {
		opcode := data[index]
		index++
		switch opcode {
		case 0x80:
			if index >= len(data) || data[index] > 5 {
				return nil, fmt.Errorf("unsupported pickle protocol")
			}
			index++
		case 0x95:
			if index+8 > len(data) {
				return nil, fmt.Errorf("truncated pickle frame")
			}
			index += 8
		case 'G':
			if index+8 > len(data) {
				return nil, fmt.Errorf("truncated pickle float")
			}
			values = append(values, math.Float64frombits(binary.BigEndian.Uint64(data[index:index+8])))
			index += 8
		case ']', 0x94, '(', 'a', 'e':
		case '.':
			return values, nil
		default:
			return nil, fmt.Errorf("unsupported pickle opcode 0x%02x", opcode)
		}
	}
	return nil, fmt.Errorf("pickle has no stop opcode")
}
