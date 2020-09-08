package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
)

type sample struct {
	level  int
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

	bReader := bufio.NewReader(os.Stdin)

	var err error
	var line []byte
	start := true

	for start && err == nil {
		line, _, err = bReader.ReadLine()
		if bytes.Contains(line, []byte("TRACE START")) {
			start = false
		}
	}

	if err != nil {
		panic(err)
	}

	ct := collapsedTrace{
		stackFreq: make(map[string]float64),
		stack:     stack{},
	}

	for err == nil {
		var isPrefix bool
		line, isPrefix, err = bReader.ReadLine()

		if isPrefix {
			for isPrefix {
				_, isPrefix, _ = bReader.ReadLine()
			}
			fmt.Println("line too long")
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
	if err != io.EOF {
		println(err)
	}
	for k, v := range ct.stackFreq {
		fmt.Printf("%s %f\n", k, v)
	}
}

func parseSample(input []byte) (sample, error) {
	var out sample
	posLevel := bytes.IndexByte(input, '\t')
	if posLevel < 0 {
		return out, errors.New("missing field")
	}
	level, err := strconv.Atoi(string(input[0:posLevel]))
	if err != nil {
		return out, err
	}
	out.level = level
	input = input[posLevel+1:]

	posFunctionNumber := bytes.IndexByte(input, '\t')
	if posFunctionNumber < 0 {
		return out, errors.New("missing field")
	}
	input = input[posFunctionNumber+1:]

	posIsExit := bytes.IndexByte(input, '\t')
	if posIsExit < 0 {
		return out, errors.New("missing field")
	}
	out.isExit = input[0] != '0'
	input = input[posIsExit+1:]

	posTime := bytes.IndexByte(input, '\t')
	if posTime < 0 {
		return out, errors.New("missing field")
	}
	time, err := strconv.ParseFloat(string(input[0:posTime]), 64)
	if err != nil {
		return out, err
	}
	out.time = time * 1_000_000
	if out.isExit {
		return out, nil
	}
	input = input[posTime+1:]

	posMemUsage := bytes.IndexByte(input, '\t')
	if posMemUsage < 0 {
		return out, errors.New("missing field")
	}
	input = input[posMemUsage+1:]

	posName := bytes.IndexByte(input, '\t')
	if posName < 0 {
		return out, errors.New("missing field")
	}
	out.name = make([]byte, posName)
	copy(out.name, input[:posName])
	return out, nil
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
