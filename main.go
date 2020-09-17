package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
)

type sample struct {
	isExit bool
	time   float64
	name   []byte
}

type stack struct {
	samples    []sample
	nameLength int
}

type collapsedTrace struct {
	stackFreq map[string]float64
	stack     stack
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}


	ct, err := collapseTrace(os.Stdin)
	if err != nil {
		panic(err)
	}
	for k, v := range ct.stackFreq {
		fmt.Printf("%s %f\n", k, v)
	}
}

func collapseTrace(r io.Reader) (collapsedTrace, error) {
	traceScanner := bufio.NewScanner(r)
	start := true

	for start && traceScanner.Scan() {
		line := traceScanner.Bytes()
		if bytes.Contains(line, []byte("TRACE START")) {
			start = false
		}
	}

	if  traceScanner.Err() != nil {
		panic(traceScanner.Err())
	}

	ct := collapsedTrace{
		stackFreq: make(map[string]float64),
		stack:     stack{},
	}

	for traceScanner.Scan() {
		line := traceScanner.Bytes()

		if line[0] == '\t' {
			break
		}

		s, errInner := parseSample(line)
		if errInner != nil {
			fmt.Fprintln(os.Stderr, "#### ERROR ####")
			fmt.Fprintln(os.Stderr, errInner.Error())
			fmt.Fprintln(os.Stderr, string(line))
			continue
		}
		ct.addSample(s)
	}
	if traceScanner.Err() != nil {
		return ct, traceScanner.Err()
	}
	return ct, nil
}

func parseSample(input []byte) (sample, error) {
	var out sample

	input = skipField(input)
	input = skipField(input)

	isExitField, input := readField(input)
	out.isExit = isExitField[0] != '0'

	timeField, input := readField(input)
	time, err := strconv.ParseFloat(string(timeField), 64)
	if err != nil {
		return out, err
	}
	out.time = time * 1_000_000
	if out.isExit {
		return out, nil
	}

	input = skipField(input)

	nameField,_ := readField(input)
	out.name = make([]byte, len(nameField))
	copy(out.name, nameField)
	return out, nil
}

func skipField(input []byte) []byte {
	nextSep := bytes.IndexByte(input, '\t')
	return input[nextSep+1:]
}

func readField(input []byte) ([]byte, []byte) {
	nextSep := bytes.IndexByte(input, '\t')
	return input[:nextSep], input[nextSep+1:]
}

func (ct *collapsedTrace) addSample(s sample) {
	if !s.isExit {
		ct.addCall(s)
		return
	}
	ct.returnFromCall(s.time)
}

func (ct *collapsedTrace) addCall(s sample) {
	ct.stack.push(s)
}

func (ct *collapsedTrace) returnFromCall(time float64) {
	if ct.stack.isEmpty() {
		return
	}
	top := ct.stack.pop()
	duration := time - top.time
	nameBuf := bytes.NewBuffer(make([]byte, 0, ct.stack.nameLength+len(top.name)+1))
	for _, v := range ct.stack.samples {
		nameBuf.Write(v.name)
		nameBuf.WriteByte(';')
	}
	nameBuf.Write(top.name)

	name := nameBuf.String()
	total, exists := ct.stackFreq[name]
	if exists {
		ct.stackFreq[name] = total + duration
	} else {
		ct.stackFreq[name] = duration
	}
}

func (s *stack) isEmpty() bool {
	return len(s.samples) == 0
}

func (s *stack) push(sample sample) {
	s.samples = append(s.samples, sample)
	s.nameLength += len(sample.name) + 1
}

func (s *stack) pop() sample {
	out := s.samples[len(s.samples)-1]
	s.samples = s.samples[:len(s.samples)-1]
	s.nameLength -= len(out.name) + 1
	return out
}
