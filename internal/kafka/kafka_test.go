package kafka

import (
	"errors"
	"testing"
)

func TestShouldRetryProduceError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unknown topic", errors.New("UNKNOWN_TOPIC_OR_PARTITION"), true},
		{"leader not available", errors.New("LEADER_NOT_AVAILABLE"), true},
		{"not leader", errors.New("NOT_LEADER_FOR_PARTITION"), true},
		{"other", errors.New("some other error"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryProduceError(tc.err); got != tc.want {
				t.Fatalf("shouldRetryProduceError()=%v want %v", got, tc.want)
			}
		})
	}
}
