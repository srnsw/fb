package main

import "testing"

func TestPack(t *testing.T) {
	if err := pack(true, true, "C:\\Users\\richardl\\Desktop\\fb-data"); err != nil {
		t.Fatal(err)
	}
}
