package norn

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"
	"strings"
)

type Command struct {
	Channel uint        `json:"channel"`
	Type    CommandType `json:"type"`
	Status  bool        `json:"status"`
}

type Response struct {
	Success    bool   `json:"isOk"`
	Message    string `json:"message"`
	Channel    uint   `json:"channel"`
	MaxChannel int    `json:"max_channel"`
	Status     bool   `json:"status"`
}

type CommandType uint

const (
	Nil         CommandType = 0
	Get         CommandType = 1
	Set         CommandType = 2
	Version     CommandType = 3
	Bootloader  CommandType = 4
	MaxChannels CommandType = 5
)

var (
	crc32c = crc32.MakeTable(crc32.Castagnoli)
	marker = []byte("E979#")
)

func protocolParser(reader *bufio.Reader) ([]byte, error) {

	markerBuffer := make([]byte, len(marker))
	if _, err := io.ReadFull(reader, markerBuffer); err != nil {
		return nil, err
	}
	if !bytes.Equal(markerBuffer, marker) {
		return nil, fmt.Errorf("invalid marker detected %s != %s", string(markerBuffer), string(marker))
	}

	const prefixLength = 9
	_, err := reader.Peek(prefixLength)
	if err != nil {
		return nil, err
	}
	lengthBuffer, err := reader.ReadString(byte('#'))
	if err != nil {
		return nil, err
	}
	lengthBuffer = strings.TrimSuffix(lengthBuffer, "#")
	mumPrefixBytes := len(lengthBuffer)
	if mumPrefixBytes != prefixLength {
		return nil, fmt.Errorf("invalid prefix length: %d expected: %d, buffer: %s", len(lengthBuffer), prefixLength, lengthBuffer)
	}

	crcBuffer := make([]byte, 8)
	if _, err := io.ReadFull(reader, crcBuffer); err != nil {
		return nil, err
	}

	crcOfInputData := crc32.Checksum([]byte(lengthBuffer), crc32c)
	crc, err := strconv.ParseUint(string(crcBuffer), 16, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to convert the CRC checksum: %w", err)
	}
	if crcOfInputData != uint32(crc) {
		return nil, fmt.Errorf("crc length field checksum do not match %08x != %08x: %w", crcOfInputData, uint32(crc), err)
	}

	numBytes, err := strconv.Atoi(lengthBuffer)
	if err != nil {
		return nil, err
	}
	dataBuffer := make([]byte, numBytes)

	if _, err := io.ReadFull(reader, dataBuffer); err != nil {
		return nil, fmt.Errorf("unable to read the data buffer: %w", err)
	}

	if _, err := io.ReadFull(reader, crcBuffer); err != nil {
		return nil, fmt.Errorf("unable to read the crc buffer: %w", err)
	}
	crcOfInputData = crc32.Checksum(dataBuffer, crc32c)
	crc, err = strconv.ParseUint(string(crcBuffer), 16, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to convert the CRC checksum: %w", err)
	}
	if crcOfInputData != uint32(crc) {
		return nil, fmt.Errorf("crc checksums do not match %08x != %08x: %w", crcOfInputData, uint32(crc), err)
	}

	return dataBuffer, nil
}

func (command *Command) Marshal() []byte {
	msg, _ := json.Marshal(command)
	length := fmt.Sprintf("%09d", len(msg))
	crcLength := fmt.Sprintf("%08x", crc32.Checksum([]byte(length), crc32c))
	payload := string(msg)
	crcPayload := fmt.Sprintf("%08x", crc32.Checksum(msg, crc32c))
	data := fmt.Sprintf("%s%s#%s%s%s", string(marker), length, crcLength, payload, crcPayload)
	fmt.Println(data)
	return []byte(data)
}

func (command *Command) Parse(reader *bufio.Reader) error {
	data, err := protocolParser(reader)
	if err != nil {
		return fmt.Errorf("error while command parsing: %w", err)
	}
	return json.Unmarshal(data, command)
}

func MarshalError(err error) string {
	res := Response{
		Success: false,
		Message: err.Error(),
	}
	return string(res.Marshal())
}

func (response *Response) Marshal() []byte {
	msg, _ := json.Marshal(response)
	length := fmt.Sprintf("%09d", len(msg))
	crcString := fmt.Sprintf("%08x", crc32.Checksum([]byte(length), crc32c))
	payload := string(msg)
	crcPayload := fmt.Sprintf("%08x", crc32.Checksum(msg, crc32c))
	data := fmt.Sprintf("%s%s#%s%s%s", string(marker), length, crcString, payload, crcPayload)
	return []byte(data)
}

func (response *Response) Parse(reader *bufio.Reader) error {
	data, err := protocolParser(reader)
	if err != nil {
		return fmt.Errorf("error while response parsing: %w", err)
	}
	return json.Unmarshal(data, response)
}
