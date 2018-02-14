package main

import "testing"

func TestFirstFunc(t *testing.T) {
	got := FirstFunc()
	if got != "123" {
		t.Errorf("Result should be 123 but was %q", got)
	}
}
