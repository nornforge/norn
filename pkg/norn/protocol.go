package norn

import (
	"bufio"
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

func protocolParser(reader *bufio.Reader) ([]byte, error) {
	const prefixLength = 9
	_, err := reader.Peek(prefixLength)
	if err != nil {
		return nil, err
	}
	lengthBuffer, err := reader.ReadString(byte(':'))
	if err != nil {
		return nil, err
	}
	lengthBuffer = strings.TrimSuffix(lengthBuffer, ":")
	mumPrefixBytes := len(lengthBuffer)
	if mumPrefixBytes != prefixLength {
		return nil, fmt.Errorf("invalid prefix length: %d expected: %d", len(lengthBuffer), prefixLength)
	}
	numBytes, err := strconv.Atoi(lengthBuffer)
	if err != nil {
		return nil, err
	}
	dataBuffer := make([]byte, numBytes)

	if _, err := io.ReadFull(reader, dataBuffer); err != nil {
		return nil, err
	}

	crcBuffer := make([]byte, 9)
	if _, err := io.ReadFull(reader, crcBuffer); err != nil {
		return nil, err
	}
	crcString := strings.TrimPrefix(string(crcBuffer), "#")
	table := crc32.MakeTable(crc32.Castagnoli)
	crcOfInputData := crc32.Checksum(dataBuffer, table)
	crc, err := strconv.ParseUint(crcString, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to convert the CRC checksum: %w", err)
	}
	if crcOfInputData != uint32(crc) {
		return nil, fmt.Errorf("crc checksums do not match %08X != %08X: %w", crcOfInputData, uint32(crc), err)
	}
	return dataBuffer, nil
}

func (command *Command) Marshal() []byte {
	table := crc32.MakeTable(crc32.Castagnoli)
	msg, _ := json.Marshal(command)
	length := fmt.Sprintf("%09d", len(msg))
	payload := string(msg)
	data := fmt.Sprintf("%s:%s", length, payload)
	crc := fmt.Sprintf("#%08x", crc32.Checksum([]byte(data), table))
	return []byte(data + crc)
}

func (command *Command) Parse(reader *bufio.Reader) error {
	data, err := protocolParser(reader)
	if err != nil {
		return err
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
	table := crc32.MakeTable(crc32.Castagnoli)
	msg, _ := json.Marshal(response)
	length := fmt.Sprintf("%09d", len(msg))
	payload := string(msg)
	data := fmt.Sprintf("%s:%s", length, payload)
	crc := fmt.Sprintf("#%08x", crc32.Checksum([]byte(data), table))
	return []byte(data + crc)
}

func (response *Response) Parse(reader *bufio.Reader) error {
	data, err := protocolParser(reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, response)
}
