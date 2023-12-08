package main

// #cgo pkg-config: gsl
// #cgo linux LDFLAGS: -lfuka_exporter -lfftw3
// #include "fukaccia.h"
import "C"

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

type BinaryType int

const (
	BNS BinaryType = iota
	BBH
	BHNS
)

func BinaryTypeToC(binaryType BinaryType) (CBinaryType C.BinaryType) {
	switch binaryType {
	case BNS:
		CBinaryType = C.BNS
	case BBH:
		CBinaryType = C.BBH
	case BHNS:
		CBinaryType = C.BHNS
	default:
		panic("unknown binary type")
	}
	return CBinaryType
}

type BinaryInfo struct {
	mass1, mass2             float64
	position_x1, position_x2 float64
}

func BinaryInfoFromC(bi C.struct_BinaryInfo) *BinaryInfo {
	return &BinaryInfo{
		mass1:       float64(bi.mass1),
		mass2:       float64(bi.mass2),
		position_x1: float64(bi.position_x1),
		position_x2: float64(bi.position_x2),
	}
}

func ReadBinaryInfo(filename string, binaryType BinaryType) *BinaryInfo {

	bi := C.read_binary_info(BinaryTypeToC(binaryType), C.CString(filename))
	return BinaryInfoFromC(bi)
}

type Grid struct {
	X, Y, Z []float64
}

type Fields struct {
}

type InterpolationRequest struct {
	BinaryType          BinaryType
	Grid                Grid
	InfoFilename        string
	InterpolationOffset float64
	InterpolationOrder  int
	RelativeDrSpacing   float64
}

func InterpolateID(req InterpolationRequest) Fields {
	pinner := &runtime.Pinner{}
	defer pinner.Unpin()

	nPoints := len(req.Grid.X)
	cgrid := C.struct_Grid{
		x:        (*C.double)(&req.Grid.X[0]),
		y:        (*C.double)(&req.Grid.Y[0]),
		z:        (*C.double)(&req.Grid.Z[0]),
		n_points: C.int(nPoints),
	}
	log.Printf("%+v", cgrid)

	gridPtr := unsafe.Pointer(&cgrid)
	pinner.Pin(gridPtr)
	creq := C.struct_FUKAInterpolateRequest{
		binary_type:          C.BHNS,
		info_filename:        C.CString(req.InfoFilename),
		grid:                 (*C.struct_Grid)(gridPtr),
		interpolation_offset: C.double(req.InterpolationOffset),
		interpolation_order:  C.int(req.InterpolationOrder),
		relative_dr_spacing:  C.double(req.RelativeDrSpacing),
	}
	creqPtr := unsafe.Pointer(&creq)
	pinner.Pin(creqPtr)
	fields := C.interpolate_FUKA_ID((*C.struct_FUKAInterpolateRequest)(creqPtr))
	v_x := unsafe.Slice(fields.v_x, nPoints)
	log.Printf("%+v", v_x)
	return Fields{}
}

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

func ReadLevelFile(filename string) (map[string][]float64, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	r := bufio.NewReader(file)

	m := map[string][]float64{}

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
		m[varname] = value
	}

	return m, nil
}

var headerString = "$BEGIN_variables:\n"

func WriteLevelFile(filename string, m map[string][]float64) error {
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
	for varname, values := range m {
		_, err = fmt.Fprintf(w, "$variable = %s : length = %d\n", varname, len(values))
		if err != nil {
			return fmt.Errorf("write header for variable %s: %w", varname, err)
		}
		err = binary.Write(w, binary.LittleEndian, values)
		if err != nil {
			return fmt.Errorf("write values for variable %s: %w", varname, err)
		}
	}

	return nil
}

func main() {
	filename := flag.String("f", "", ".info file")
	flag.Parse()

	m, err := ReadLevelFile("level3.173")
	if err != nil {
		log.Fatal(err)
	}

	err = WriteLevelFile("outlevel", m)
	if err != nil {
		log.Fatal(err)
	}

	bi := ReadBinaryInfo(*filename, BHNS)
	log.Printf("%+v", bi)

	nPoints := 1024
	x := make([]float64, nPoints)

	grid := Grid{
		X: x,
		Y: make([]float64, nPoints),
		Z: make([]float64, nPoints),
	}

	req := InterpolationRequest{
		BinaryType:          BHNS,
		InfoFilename:        *filename,
		Grid:                grid,
		InterpolationOffset: 0.0,
		InterpolationOrder:  8,
		RelativeDrSpacing:   0.3,
	}
	_ = InterpolateID(req)
}
