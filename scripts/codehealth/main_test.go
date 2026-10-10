package main

import "testing"

func TestFunctions(t *testing.T) {
	source := []byte(`package p
type Box[T any] struct{}
func (b *Box[T]) Run(a, b bool) { if a && b || a { }; for { break }; for range []int{} {}; switch { case a: case b: default: }; select { case <-make(chan int): default: }; f := func(){ if a {} }; _ = f }
func Empty() {}
`)
	rows, err := functions("internal/p/p.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != "internal/p/p.go:*Box[T].Run" || rows[0].Complexity != 10 || rows[1].Complexity != 1 {
		t.Fatalf("unexpected: %+v", rows)
	}
	if _, err := functions("broken.go", []byte("not Go")); err == nil {
		t.Fatal("accepted invalid syntax")
	}
}
