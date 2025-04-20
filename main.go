package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// FileHeader is the on-disk header for a SYMF file:
//
//	Magic       uint32 ("SYMF" == 0x46594D53)
//	SymbolCount uint32
//	StringsSize uint64
//
// Total size: 16 bytes.
type FileHeader struct {
	Magic       uint32
	SymbolCount uint32
	StringsSize uint64
}

// Symbol is each fixed-size entry in the SYMF file:
//
//	Address    uint64
//	Type       byte   ('T' or 't')
//	NameOffset uint64 (offset into the string table blob)
//
// Total size written: 8 + 1 + 8 = 17 bytes per entry.
type Symbol struct {
	Address    uint64
	Type       byte
	NameOffset uint64
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <object-file> <out.symf>\n", os.Args[0])
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}
	objPath := flag.Arg(0)
	outPath := flag.Arg(1)
	
	out, err := exec.Command("nm", objPath).Output()
	if err != nil {
		log.Fatalf("failed to run nm on %s: %v", objPath, err)
	}

	type rawSym struct {
		addr uint64
		typ  byte
		name string
	}
	var raws []rawSym
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		t := fields[1]
		if t != "T" && t != "t" {
			continue
		}
		// Parse address as hex
		a, err := strconv.ParseUint(fields[0], 16, 64)
		if err != nil {
			log.Fatalf("invalid address %q: %v", fields[0], err)
		}
		raws = append(raws, rawSym{addr: a, typ: t[0], name: fields[2]})
	}

	var strtab bytes.Buffer
	syms := make([]Symbol, len(raws))
	for i, rs := range raws {
		syms[i].Address = rs.addr
		syms[i].Type = rs.typ
		syms[i].NameOffset = uint64(strtab.Len())
		strtab.WriteString(rs.name)
		strtab.WriteByte(0) // null-terminate
	}

	const symfMagic = 0x464D5953
	head := FileHeader{
		Magic:       symfMagic,
		SymbolCount: uint32(len(syms)),
		StringsSize: uint64(strtab.Len()),
	}

	// 5) Write everything out
	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("cannot create %s: %v", outPath, err)
	}
	defer f.Close()

	if err := binary.Write(f, binary.LittleEndian, head); err != nil {
		log.Fatalf("failed to write header: %v", err)
	}
	for _, e := range syms {
		if err := binary.Write(f, binary.LittleEndian, e); err != nil {
			log.Fatalf("failed to write symbol entry: %v", err)
		}
	}
	if _, err := f.Write(strtab.Bytes()); err != nil {
		log.Fatalf("failed to write string table: %v", err)
	}

	log.Printf("Wrote %d symbols (%d bytes of names) to %s\n",
		len(syms), strtab.Len(), outPath)
}
