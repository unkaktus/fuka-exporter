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

func unsafeDoubleSlice(ptr *C.double, clen C.int) []float64 {
	len := int(clen)
	doubles := unsafe.Slice(ptr, len)
	floats := make([]float64, len)
	for i, d := range doubles {
		floats[i] = float64(d)
	}
	return floats
}

type Grid struct {
	X, Y, Z []float64
}

type Fields struct {
	alpha []float64
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
	cfields := C.interpolate_FUKA_ID((*C.struct_FUKAInterpolateRequest)(creqPtr))

	fields := Fields{
		alpha: unsafeDoubleSlice(cfields.alpha, C.int(nPoints)),
	}
	return fields
}

//export fukaccia_interpolate
func fukaccia_interpolate(creq *C.struct_FUKAInterpolateRequest, n_workers C.int) C.struct_Fields {
	log.Printf("fukaccia: requested %d workers", int(n_workers))
	grid := Grid{
		X: unsafeDoubleSlice(creq.grid.x, creq.grid.n_points)[:100],
		Y: unsafeDoubleSlice(creq.grid.y, creq.grid.n_points)[:100],
		Z: unsafeDoubleSlice(creq.grid.z, creq.grid.n_points)[:100],
	}

	log.Printf("grid: %v points, (%v, %v)", len(grid.X), grid.X[0], grid.X[len(grid.X)-1])
	req := InterpolationRequest{
		BinaryType:          BHNS,
		InfoFilename:        C.GoString(creq.info_filename),
		Grid:                grid,
		InterpolationOffset: 0.0,
		InterpolationOrder:  8,
		RelativeDrSpacing:   0.3,
	}
	fields := InterpolateID(req)

	log.Printf("alpha: %v points, (%v, %v)", len(fields.alpha), fields.alpha[0], fields.alpha[len(fields.alpha)-1])

	return C.struct_Fields{}
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
