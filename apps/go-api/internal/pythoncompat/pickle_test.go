package pythoncompat

import (
	"encoding/hex"
	"math"
	"reflect"
	"testing"
)

func TestFloatListPickleProtocolFourGoldenForms(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		wire   string
	}{
		{name: "empty", values: []float64{}, wire: "80049503000000000000005d942e"},
		{name: "single", values: []float64{1.5}, wire: "8004950d000000000000005d94473ff8000000000000612e"},
		{name: "multi with timestamps", values: []float64{3, 2, 1}, wire: "80049520000000000000005d9428474008000000000000474000000000000000473ff0000000000000652e"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wire, err := hex.DecodeString(test.wire)
			if err != nil {
				t.Fatal(err)
			}
			if got := EncodeFloatListPickle(test.values); !reflect.DeepEqual(got, wire) {
				t.Fatalf("encoded=%x want=%x", got, wire)
			}
			got, err := DecodeFloatListPickle(wire)
			if err != nil || !reflect.DeepEqual(got, test.values) {
				t.Fatalf("decoded=%v err=%v want=%v", got, err, test.values)
			}
		})
	}
}

func TestDecodeFloatListPickleRejectsMalformedStreams(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
		want string
	}{
		{name: "unsupported protocol", wire: []byte{0x80, 0x06}, want: "unsupported pickle protocol"},
		{name: "truncated frame", wire: []byte{0x95, 1}, want: "truncated pickle frame"},
		{name: "truncated float", wire: []byte{'G', 0}, want: "truncated pickle float"},
		{name: "unsupported opcode", wire: []byte{'x'}, want: "unsupported pickle opcode 0x78"},
		{name: "missing stop", wire: []byte{']', 0x94}, want: "pickle has no stop opcode"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if values, err := DecodeFloatListPickle(test.wire); err == nil || err.Error() != test.want {
				t.Fatalf("decoded malformed stream as %v, err=%v; want error %q", values, err, test.want)
			}
		})
	}
}

func TestDecodeFloatListPickleStopsAtFirstStopOpcode(t *testing.T) {
	encoded := EncodeFloatListPickle([]float64{math.Pi})
	encoded = append(encoded, 'x')
	values, err := DecodeFloatListPickle(encoded)
	if err != nil || !reflect.DeepEqual(values, []float64{math.Pi}) {
		t.Fatalf("decoded=%v err=%v", values, err)
	}
}
