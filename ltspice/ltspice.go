package ltspice

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

type Simulation struct {
	MetaData *RawFileMetadata
	Data     map[string][]float64
}

func Parse(fileName string) (*Simulation, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	meta, err := parseHeaders(reader)
	if err != nil {
		return nil, err
	}
	data, err := parseBinaryData(reader, meta)
	if err != nil {
		return nil, err
	}
	sim := &Simulation{
		MetaData: meta,
		Data:     data,
	}
	return sim, nil
}

func readLineUTF16(r io.Reader) (string, error) {
	lineBuff := make([]uint16, 0, maxLineSize)
	buff := make([]byte, 2)
	for {
		if len(lineBuff) > maxLineSize {
			return "", ErrLineTooLong
		}

		_, err := io.ReadFull(r, buff)

		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", ErrUnexpectedEndOfFile
			} else {
				return "", ErrUnexpectedError
			}
		}
		rune := binary.LittleEndian.Uint16(buff)
		if rune == '\n' {
			return string(utf16.Decode(lineBuff)), nil
		}

		lineBuff = append(lineBuff, rune)
	}
}

func parseHeaders(reader io.Reader) (*RawFileMetadata, error) {
	var metadata = &RawFileMetadata{Flags: None}
	for {
		line, err := readLineUTF16(reader)
		if err != nil {
			return nil, err
		}
		if strings.Contains(strings.ToLower(strings.TrimSpace(line)), HeaderBinary) || strings.Contains(strings.ToLower(strings.TrimSpace(line)), HeaderValues) {
			break
		}
		err = parseHeaderLine(reader, metadata, line)
		if err != nil {
			return nil, err
		}
	}
	return metadata, nil
}

func parseBinaryData(reader io.Reader, meta *RawFileMetadata) (map[string][]float64, error) {
	data := make(map[string][]float64)
	for _, v := range meta.Variables {
		data[v.Name] = make([]float64, meta.NoPoints)
	}
	buff := make([]byte, 16)
	for i := 0; i < meta.NoPoints; i++ {
		for _, v := range meta.Variables {
			_, err := io.ReadFull(reader, buff[:v.Size])
			if err != nil {
				fmt.Println(err.Error())
				return nil, err
			}
			var val float64
			if v.Size == 4 {
				val = toFloatFrom32(buff[:v.Size])
			} else {
				val = toFloat(buff[:v.Size])
			}
			data[v.Name][i] = val
		}
	}
	return data, nil
}
