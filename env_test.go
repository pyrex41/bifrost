package main

import (
	"reflect"
	"testing"
)

func TestExpandPortSelection(t *testing.T) {
	known := []string{"shen-cl", "shen-go", "shen-joy"}
	got, err := expandPortSelection([]string{"shen-joy", "shen-go", "shen-joy"}, known)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"shen-joy", "shen-go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selection = %v, want %v", got, want)
	}
	got, err = expandPortSelection([]string{"all"}, known)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, known) {
		t.Fatalf("all = %v, want %v", got, known)
	}
}

func TestExpandPortSelectionRejectsUnknown(t *testing.T) {
	if _, err := expandPortSelection([]string{"shen-missing"}, []string{"shen-go"}); err == nil {
		t.Fatal("unknown port was accepted")
	}
}
