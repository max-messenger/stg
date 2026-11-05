package domain

import "testing"

func TestCallback(t *testing.T) {
	cb := CallBack{
		InternalID: 5,
		Key:        "key",
	}

	if cb.String() != "key:5" {
		t.Errorf("callback.String() expected %s, got %s", "key", cb.String())
	}
}
