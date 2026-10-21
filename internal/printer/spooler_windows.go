//go:build windows

package printer

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

var (
	winspool          = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter   = winspool.NewProc("OpenPrinterW")
	procClosePrinter  = winspool.NewProc("ClosePrinter")
	procStartDocPrint = winspool.NewProc("StartDocPrinterW")
	procEndDocPrint   = winspool.NewProc("EndDocPrinter")
	procStartPage     = winspool.NewProc("StartPagePrinter")
	procEndPage       = winspool.NewProc("EndPagePrinter")
	procWritePrinter  = winspool.NewProc("WritePrinter")
)

type docInfo1 struct {
	pDocName    *uint16
	pOutputFile *uint16
	pDatatype   *uint16
}

// SpoolerPrinter mengimplementasikan print via Windows Spooler RAW
type SpoolerPrinter struct {
	printerName string
	mu          sync.Mutex
}

func NewSpoolerPrinter(printerName string) (Printer, error) {
	if printerName == "" {
		printerName = "EPSON PLQ-20 ESC/P2"
	}
	return &SpoolerPrinter{
		printerName: printerName,
	}, nil
}

func (s *SpoolerPrinter) Name() string {
	return s.printerName
}

func (s *SpoolerPrinter) Ping() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	namePtr, err := syscall.UTF16PtrFromString(s.printerName)
	if err != nil {
		return false, err
	}

	var hPrinter syscall.Handle
	r1, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		return false, fmt.Errorf("printer Windows '%s' tidak dapat dibuka: %w", s.printerName, err)
	}

	procClosePrinter.Call(uintptr(hPrinter))
	return true, nil
}

func (s *SpoolerPrinter) Print(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	namePtr, err := syscall.UTF16PtrFromString(s.printerName)
	if err != nil {
		return err
	}

	var hPrinter syscall.Handle
	r1, _, err := procOpenPrinter.Call(
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(&hPrinter)),
		0,
	)
	if r1 == 0 {
		return fmt.Errorf("printer Windows '%s' tidak terdeteksi atau mati: %w", s.printerName, err)
	}
	defer procClosePrinter.Call(uintptr(hPrinter))

	docName, _ := syscall.UTF16PtrFromString("Passbook Print Job")
	dataType, _ := syscall.UTF16PtrFromString("RAW")

	di := docInfo1{
		pDocName:    docName,
		pOutputFile: nil,
		pDatatype:   dataType,
	}

	docJob, _, err := procStartDocPrint.Call(
		uintptr(hPrinter),
		1,
		uintptr(unsafe.Pointer(&di)),
	)
	if docJob == 0 {
		return fmt.Errorf("gagal memulai print job (StartDocPrinter): %w", err)
	}
	defer procEndDocPrint.Call(uintptr(hPrinter))

	rPage, _, err := procStartPage.Call(uintptr(hPrinter))
	if rPage == 0 {
		return fmt.Errorf("gagal memulai halaman printer (StartPagePrinter): %w", err)
	}

	var written uint32
	rWrite, _, err := procWritePrinter.Call(
		uintptr(hPrinter),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)),
	)
	if rWrite == 0 {
		procEndPage.Call(uintptr(hPrinter))
		return fmt.Errorf("gagal mengirim bytes ke printer spooler (WritePrinter): %w", err)
	}

	procEndPage.Call(uintptr(hPrinter))
	return nil
}

func (s *SpoolerPrinter) Close() error {
	return nil
}
