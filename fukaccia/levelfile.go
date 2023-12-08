package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func ParseVarHeader(h string) (string, int, error) {
	h = strings.TrimRight(h, "\n")
	sp := strings.Split(h, " : ")
	varnameKV := strings.Split(sp[0], " = ")
	if len(varnameKV) != 2 {
		return "", 0, fmt.Errorf("malformed header")
	}
	lengthKV := strings.Split(sp[1], " = ")
	if len(lengthKV) != 2 {
		return "", 0, fmt.Errorf("malformed header")
	}
	varname := varnameKV[1]
	length, err := strconv.Atoi(lengthKV[1])
	if err != nil {
		return "", 0, fmt.Errorf("convert length to int")
	}
	return varname, length, nil

}

func ReadVariable(r *bufio.Reader) (string, []float64, error) {
	varString, err := r.ReadString('\n')
	if err != nil {
		return "", nil, fmt.Errorf("read var string: %w", err)
	}
	varname, length, err := ParseVarHeader(varString)
	if err != nil {
		return "", nil, fmt.Errorf("parse variable header: %w", err)
	}
	v := make([]float64, length)
	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return "", nil, fmt.Errorf("read binary: %w", err)
	}
	return varname, v, nil
}

type LevelFile struct {
	m    map[string][]float64
	keys []string
}

func ReadLevelFile(filename string) (*LevelFile, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	r := bufio.NewReader(file)

	lf := &LevelFile{
		m: map[string][]float64{},
	}

	for {
		_, err = r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read begin")
		}
		varname, value, err := ReadVariable(r)
		if errors.Unwrap(err) == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		lf.m[varname] = value
		lf.keys = append(lf.keys, varname)
	}

	return lf, nil
}

var headerString = "$BEGIN_variables:\n"

func WriteLevelFile(filename string, lf *LevelFile) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	w := bufio.NewWriter(file)
	defer w.Flush()

	_, err = w.WriteString(headerString)
	if err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, varname := range lf.keys {
		values := lf.m[varname]
		_, err = fmt.Fprintf(w, "$variable = %s : length = %d\n", varname, len(values))
		if err != nil {
			return fmt.Errorf("write header for variable %s: %w", varname, err)
		}
		err = binary.Write(w, binary.LittleEndian, values)
		if err != nil {
			return fmt.Errorf("write values for variable %s: %w", varname, err)
		}
		_, err = w.WriteString("\n")
		if err != nil {
			return fmt.Errorf("write trailing newline: %w", err)
		}
	}

	return nil
}
