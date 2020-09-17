package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"math"
	"testing"
)

func TestParseLine(t *testing.T) {
	type testCase struct {
		line     []byte
		expected sample
	}
	cases := map[string]testCase{
		"function entry line": {
			[]byte("2\t4\t0\t0.000312\t434896\trequire\t1\t/app/vendor/autoload.php\t/app/public/index.php\t32"),
			sample{
				isExit: false,
				name:   []byte("require"),
				time:   312.0,
			},
		},
		"function exit line": {
			[]byte("3\t5\t1\t0.000360\t435784"),
			sample{
				isExit: true,
				time:   360.0,
			},
		},
	}

	for name, caseData := range cases {
		line, expected := caseData.line, caseData.expected
		t.Log(name)
		s, err := parseSample(line)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if s.isExit != expected.isExit {
			t.Errorf("expected s.isExit to equal %v got %v", expected.isExit, s.isExit)
		}
		if string(s.name) != string(expected.name) {
			t.Errorf("expected s.name to equal %q got %q", expected.name, s.name)
		}
		if s.time != expected.time {
			t.Errorf("expected s.name to equal %v got %v", expected.time, s.time)
		}
	}
}

func TestCollapseTrace(t *testing.T) {
	data, err := ioutil.ReadFile("trace.test.xt")
	if err != nil {
		t.Fatal(err)
	}
	ct, err := collapseTrace(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	assertStackHasDuration("{main};a;c", 140.000000, ct, t)
	assertStackHasDuration("{main};a", 189.000000, ct, t)
	assertStackHasDuration("{main};b;d;e", 18.000000, ct, t)
	assertStackHasDuration("{main};b;d", 39.000000, ct, t)
	assertStackHasDuration("{main};b;f", 7.000000, ct, t)
	assertStackHasDuration("{main};b", 68.000000, ct, t)
	assertStackHasDuration("{main}", 302.000000, ct, t)
}

func assertStackHasDuration(stackName string, duration float64, ct collapsedTrace, t *testing.T) {
	const delta = 0.0001
	stackcount, ok := ct.stackFreq[stackName]
	if !ok {
		t.Errorf("no entry for %q", stackName)
		return
	}
	if math.Abs(stackcount-duration) > delta {
		t.Errorf("missing or wrong entry for stack2: %q got: %f", stackName, stackcount)
	}
}
