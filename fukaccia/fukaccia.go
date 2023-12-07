package main

// #cgo pkg-config: gsl
// #cgo linux LDFLAGS: -lfuka_exporter -lfftw3
// #include "fukaccia.h"
import "C"

import (
	"flag"
	"log"
	"runtime"
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

func main() {
	filename := flag.String("f", "", ".info file")
	flag.Parse()

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
