package main

import (
	"testing"
	"time"
)

func TestBootstrapTimeout(t *testing.T) {
	for _, tt := range []struct {
		value   string
		want    time.Duration
		invalid bool
	}{
		{"", time.Hour, false}, {"4h", 4 * time.Hour, false},
		{"0", 0, true}, {"-1s", 0, true}, {"invalid", 0, true},
	} {
		t.Run(tt.value, func(t *testing.T) {
			got, err := bootstrapTimeout(tt.value)
			if (err != nil) != tt.invalid || got != tt.want {
				t.Fatalf("timeout=%v error=%v", got, err)
			}
		})
	}
}
