package main

// #cgo pkg-config: gsl
// #cgo linux LDFLAGS: -lfuka_exporter -lfftw3
// #include "fukaccia.h"
import "C"

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"unsafe"
)

var logger = log.New(os.Stderr, "fukaccia: ", 0)

var pinner = &runtime.Pinner{}

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

func toDoubleArray(x []float64) *C.double {
	cx := (*C.double)(unsafe.Pointer(&x[0]))
	pinner.Pin(cx)
	return cx
}

type Grid struct {
	X, Y, Z []float64
}

func (grid *Grid) Split(n, nChunks int) Grid {
	chunkSize := len(grid.X) / nChunks
	offset := chunkSize * n
	if n == nChunks-1 {
		chunkSize += len(grid.X) % nChunks
	}
	return Grid{
		X: grid.X[offset : offset+chunkSize],
		Y: grid.Y[offset : offset+chunkSize],
		Z: grid.Z[offset : offset+chunkSize],
	}
}

type Fields map[string][]float64

func FieldsFromC(cfields C.struct_Fields, nPoints int) (fields Fields) {
	fields = Fields{
		"alpha":  unsafeDoubleSlice(cfields.alpha, C.int(nPoints)),
		"beta_x": unsafeDoubleSlice(cfields.beta_x, C.int(nPoints)),
		"beta_y": unsafeDoubleSlice(cfields.beta_y, C.int(nPoints)),
		"beta_z": unsafeDoubleSlice(cfields.beta_z, C.int(nPoints)),

		"gamma_xx": unsafeDoubleSlice(cfields.gamma_xx, C.int(nPoints)),
		"gamma_xy": unsafeDoubleSlice(cfields.gamma_xy, C.int(nPoints)),
		"gamma_xz": unsafeDoubleSlice(cfields.gamma_xz, C.int(nPoints)),
		"gamma_yy": unsafeDoubleSlice(cfields.gamma_yy, C.int(nPoints)),
		"gamma_yz": unsafeDoubleSlice(cfields.gamma_yz, C.int(nPoints)),
		"gamma_zz": unsafeDoubleSlice(cfields.gamma_zz, C.int(nPoints)),

		"K_xx": unsafeDoubleSlice(cfields.K_xx, C.int(nPoints)),
		"K_xy": unsafeDoubleSlice(cfields.K_xy, C.int(nPoints)),
		"K_xz": unsafeDoubleSlice(cfields.K_xz, C.int(nPoints)),
		"K_yy": unsafeDoubleSlice(cfields.K_yy, C.int(nPoints)),
		"K_yz": unsafeDoubleSlice(cfields.K_yz, C.int(nPoints)),
		"K_zz": unsafeDoubleSlice(cfields.K_zz, C.int(nPoints)),

		"rho":      unsafeDoubleSlice(cfields.rho, C.int(nPoints)),
		"epsilon":  unsafeDoubleSlice(cfields.epsilon, C.int(nPoints)),
		"pressure": unsafeDoubleSlice(cfields.pressure, C.int(nPoints)),
		"v_x":      unsafeDoubleSlice(cfields.v_x, C.int(nPoints)),
		"v_y":      unsafeDoubleSlice(cfields.v_y, C.int(nPoints)),
		"v_z":      unsafeDoubleSlice(cfields.v_z, C.int(nPoints)),
	}
	return fields
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
	nPoints := len(req.Grid.X)
	cgrid := C.struct_Grid{
		x:        (*C.double)(&req.Grid.X[0]),
		y:        (*C.double)(&req.Grid.Y[0]),
		z:        (*C.double)(&req.Grid.Z[0]),
		n_points: C.int(nPoints),
	}

	gridPtr := unsafe.Pointer(&cgrid)
	pinner.Pin(gridPtr)
	creq := C.struct_FUKAInterpolateRequest{
		binary_type:          BinaryTypeToC(req.BinaryType),
		info_filename:        C.CString(req.InfoFilename),
		grid:                 (*C.struct_Grid)(gridPtr),
		interpolation_offset: C.double(req.InterpolationOffset),
		interpolation_order:  C.int(req.InterpolationOrder),
		relative_dr_spacing:  C.double(req.RelativeDrSpacing),
	}
	creqPtr := unsafe.Pointer(&creq)
	pinner.Pin(creqPtr)
	cfields := C.interpolate_FUKA_ID((*C.struct_FUKAInterpolateRequest)(creqPtr))
	fields := FieldsFromC(cfields, nPoints)
	return fields
}

func InterpolateID_worker(req *InterpolationRequest) (*Fields, error) {
	cmd := exec.Command("fukaccia", "interpolate")
	cmd.Stderr = os.Stderr
	stdin := bytes.NewBuffer(nil)
	cmd.Stdin = stdin

	enc := gob.NewEncoder(stdin)
	if err := enc.Encode(*req); err != nil {
		return nil, fmt.Errorf("cannot encode: %w", err)
	}
	stdout := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cannot run command: %w", err)
	}
	dec := gob.NewDecoder(stdout)
	fields := &Fields{}
	if err := dec.Decode(fields); err != nil {
		return nil, fmt.Errorf("cannot decode: %w", err)
	}
	return fields, nil
}

//export fukaccia_interpolate
func fukaccia_interpolate(creq *C.struct_FUKAInterpolateRequest, n_workers C.int) C.struct_Fields {
	nWorkers := int(n_workers)
	logger.Printf("requested %d workers", int(n_workers))
	grid := Grid{
		X: unsafeDoubleSlice(creq.grid.x, creq.grid.n_points),
		Y: unsafeDoubleSlice(creq.grid.y, creq.grid.n_points),
		Z: unsafeDoubleSlice(creq.grid.z, creq.grid.n_points),
	}

	// logger.Printf("grid: %v points, (%v, %v)", len(grid.X), grid.X[0], grid.X[len(grid.X)-1])

	var wg sync.WaitGroup

	fieldsChunks := make([]Fields, nWorkers)
	var fieldsChunksLock sync.Mutex

	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chunkGrid := grid.Split(i, nWorkers)
			logger.Printf("chunkGrid #%d: %v points, (%v, %v)", i, len(chunkGrid.X), chunkGrid.X[0], chunkGrid.X[len(chunkGrid.X)-1])
			req := &InterpolationRequest{
				BinaryType:          BinaryType(creq.binary_type),
				InfoFilename:        C.GoString(creq.info_filename),
				Grid:                chunkGrid,
				InterpolationOffset: float64(creq.interpolation_offset),
				InterpolationOrder:  int(creq.interpolation_order),
				RelativeDrSpacing:   float64(creq.relative_dr_spacing),
			}
			fields, err := InterpolateID_worker(req)
			if err != nil {
				logger.Fatalf("interpolation worker #%d: %v", i, err)
			}
			fieldsChunksLock.Lock()
			fieldsChunks[i] = *fields
			fieldsChunksLock.Unlock()
			// logger.Printf("decoded fields: %v", fields)
		}(i)
	}
	wg.Wait()
	fields := fieldsChunks[0]
	for i := 1; i < len(fieldsChunks); i++ {
		for k, v := range fieldsChunks[i] {
			fields[k] = append(fields[k], v...)
		}
	}

	// logger.Printf("done with interpolation, yay!")
	// logger.Printf("full fields: %+v", fields)
	// logger.Printf("alpha: %v points, (%v, %v)", len(fields["alpha"]), fields["alpha"][0], fields["alpha"][len(fields["alpha"])-1])

	cfields := C.struct_Fields{
		alpha: toDoubleArray(fields["alpha"]),

		beta_x: toDoubleArray(fields["beta_x"]),
		beta_y: toDoubleArray(fields["beta_y"]),
		beta_z: toDoubleArray(fields["beta_z"]),

		gamma_xx: toDoubleArray(fields["gamma_xx"]),
		gamma_xy: toDoubleArray(fields["gamma_xy"]),
		gamma_xz: toDoubleArray(fields["gamma_xz"]),
		gamma_yy: toDoubleArray(fields["gamma_yy"]),
		gamma_yz: toDoubleArray(fields["gamma_yz"]),
		gamma_zz: toDoubleArray(fields["gamma_zz"]),

		K_xx: toDoubleArray(fields["K_xx"]),
		K_xy: toDoubleArray(fields["K_xy"]),
		K_xz: toDoubleArray(fields["K_xz"]),
		K_yy: toDoubleArray(fields["K_yy"]),
		K_yz: toDoubleArray(fields["K_yz"]),
		K_zz: toDoubleArray(fields["K_zz"]),

		rho:      toDoubleArray(fields["rho"]),
		epsilon:  toDoubleArray(fields["epsilon"]),
		pressure: toDoubleArray(fields["pressure"]),
		v_x:      toDoubleArray(fields["v_x"]),
		v_y:      toDoubleArray(fields["v_y"]),
		v_z:      toDoubleArray(fields["v_z"]),
	}

	return cfields
}

//export fukaccia_finalize
func fukaccia_finalize() {
	pinner.Unpin()
}

func main() {
	flag.Parse()
	if flag.Arg(0) != "interpolate" {
		logger.Fatal("unknown action")
	}
	dec := gob.NewDecoder(os.Stdin)
	req := InterpolationRequest{}
	err := dec.Decode(&req)
	if err != nil {
		logger.Fatalf("decode interpolation request: %v", err)
	}
	fields := InterpolateID(req)

	enc := gob.NewEncoder(os.Stdout)
	if err := enc.Encode(fields); err != nil {
		logger.Fatalf("encode fields: %v", err)
	}
}
