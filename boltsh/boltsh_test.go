package main

import "testing"

func TestEmpty(t *testing.T) {
	res := argSplit("")
	if len(res) != 0 {
		t.Errorf("Expected empty slice for empty string, got %v\n", res)
	}
}

func assertLen(t *testing.T, expected int, result []string) {
	if len(result) != expected {
		t.Errorf("Expected slice of size %v, got %v\n", expected, result)
	}
}

func assertEquals(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected %v, got %v\n", expected, actual)
	}
}

func TestSingle(t *testing.T) {
	value := "one"
	res := argSplit(value)
	assertLen(t, 1, res)
	assertEquals(t, value, res[0])
}

func TestMulti(t *testing.T) {
	line := " one\ttwo  three"
	res := argSplit(line)
	assertLen(t, 3, res)
	assertEquals(t, "one", res[0])
	assertEquals(t, "two", res[1])
	assertEquals(t, "three", res[2])
}

func TestSingleQuote(t *testing.T) {
	line := " \"all as one\" "
	res := argSplit(line)
	assertLen(t, 1, res)
	assertEquals(t, "all as one", res[0])
}

func TestMixed(t *testing.T) {
	line := "single \"all as one\" single"
	res := argSplit(line)
	assertLen(t, 3, res)
	assertEquals(t, "single", res[0])
	assertEquals(t, "all as one", res[1])
	assertEquals(t, "single", res[2])
}

func TestMissingSpace(t *testing.T) {
	line := "where do\"I begin?\""
	res := argSplit(line)
	assertLen(t, 2, res)
	assertEquals(t, "where", res[0])
	assertEquals(t, "doI begin?", res[1])
}

func TestUnterm(t *testing.T) {
	line := "\"do I end?"
	res := argSplit(line)
	assertLen(t, 1, res)
	assertEquals(t, "do I end?", res[0])
}
