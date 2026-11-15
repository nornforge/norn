package norn

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"strconv"
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
	// Brightness sets the brightness of the status LED of the device. The
	// value, from 0 (LED off) to MaxBrightness, is carried in the Channel
	// field of the command and echoed in the Channel field of the response.
	Brightness CommandType = 6
)

// MaxBrightness is the highest value a Brightness command accepts.
const MaxBrightness = 255

var (
	crc32c = crc32.MakeTable(crc32.Castagnoli)
	marker = []byte("E979#")
)

// findMarker reads from the reader until it has consumed a complete marker.
// Bytes before the marker are skipped: after a frame was cut off or garbled,
// the parser picks up again at the start of the next frame instead of
// staying out of sync with the sender.
func findMarker(reader *bufio.Reader) error {
	matched := 0
	for matched < len(marker) {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}
		switch b {
		case marker[matched]:
			matched++
		case marker[0]:
			// The marker does not repeat its first byte, so a mismatch
			// can only be the start of a new marker.
			matched = 1
		default:
			matched = 0
		}
	}
	return nil
}

const (
	lengthDigits = 9                    // digits of the length field
	headerLength = lengthDigits + 1 + 8 // length field, '#', checksum of the length field
	crcLength    = 8                    // checksum of the payload
)

// parseChecksum checks that crcField holds the checksum of data.
func parseChecksum(crcField, data []byte) error {
	crc, err := strconv.ParseUint(string(crcField), 16, 32)
	if err != nil {
		return fmt.Errorf("unable to convert the CRC checksum: %w", err)
	}
	if sum := crc32.Checksum(data, crc32c); sum != uint32(crc) {
		return fmt.Errorf("%08x != %08x", sum, uint32(crc))
	}
	return nil
}

// parseHeader returns the payload length the header after the marker announces.
func parseHeader(header []byte) (int, error) {
	if header[lengthDigits] != '#' {
		return 0, fmt.Errorf("invalid prefix length: expected %d digits followed by '#', buffer: %q", lengthDigits, header[:lengthDigits+1])
	}
	lengthField := header[:lengthDigits]
	if err := parseChecksum(header[lengthDigits+1:headerLength], lengthField); err != nil {
		return 0, fmt.Errorf("crc length field checksum do not match: %w", err)
	}
	return strconv.Atoi(string(lengthField))
}

// peekOrRead returns the next n bytes without consuming them, or consumes
// them if the buffer of the reader is too small to peek that far.
func peekOrRead(reader *bufio.Reader, n int) ([]byte, error) {
	if n <= reader.Size() {
		return reader.Peek(n)
	}
	buffer := make([]byte, n)
	_, err := io.ReadFull(reader, buffer)
	return buffer, err
}

// protocolParser reads the next frame and returns its payload.
//
// Bytes before a marker are skipped. When a frame turns out to be invalid,
// only its marker is consumed, as long as the frame fits into the buffer of
// the reader: the next call looks for a marker in the bytes after it. A frame
// that was cut off then does not take the frame that follows it down too.
func protocolParser(reader *bufio.Reader) ([]byte, error) {
	if err := findMarker(reader); err != nil {
		return nil, err
	}
	header, err := peekOrRead(reader, headerLength)
	if err != nil {
		return nil, err
	}
	numBytes, err := parseHeader(header)
	if err != nil {
		return nil, err
	}

	frameLength := headerLength + numBytes + crcLength
	if frameLength <= reader.Size() {
		frame, err := reader.Peek(frameLength)
		if err != nil {
			return nil, fmt.Errorf("unable to read the data buffer: %w", err)
		}
		data := frame[headerLength : headerLength+numBytes]
		if err := parseChecksum(frame[headerLength+numBytes:], data); err != nil {
			return nil, fmt.Errorf("crc checksums do not match %w", err)
		}
		data = append([]byte(nil), data...) // Peek's slice is only valid until the next read
		_, err = reader.Discard(frameLength)
		return data, err
	}

	// The frame is larger than the buffer, so it is read the plain way.
	if headerLength <= reader.Size() {
		if _, err := reader.Discard(headerLength); err != nil {
			return nil, err
		}
	}
	data := make([]byte, numBytes)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("unable to read the data buffer: %w", err)
	}
	crcField := make([]byte, crcLength)
	if _, err := io.ReadFull(reader, crcField); err != nil {
		return nil, fmt.Errorf("unable to read the crc buffer: %w", err)
	}
	if err := parseChecksum(crcField, data); err != nil {
		return nil, fmt.Errorf("crc checksums do not match %w", err)
	}
	return data, nil
}

func (command *Command) Marshal() []byte {
	msg, _ := json.Marshal(command)
	length := fmt.Sprintf("%09d", len(msg))
	crcLength := fmt.Sprintf("%08x", crc32.Checksum([]byte(length), crc32c))
	payload := string(msg)
	crcPayload := fmt.Sprintf("%08x", crc32.Checksum(msg, crc32c))
	data := fmt.Sprintf("%s%s#%s%s%s", string(marker), length, crcLength, payload, crcPayload)
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

// Answers reports whether the response can be the answer to the command.
// The protocol carries no request id, but the device echoes the channel of
// Get, Set and Brightness commands, and only answers Version and MaxChannels
// with a version and a channel count. A host uses this to skip a response
// that belongs to an earlier command, for example one that arrived after the
// host had given up waiting for it.
//
// A failed response carries nothing to compare, so it counts as an answer to
// any command.
func (response *Response) Answers(command Command) bool {
	if !response.Success {
		return true
	}
	switch command.Type {
	case Get, Set, Brightness:
		return response.Channel == command.Channel && response.MaxChannel == 0 && response.Message == ""
	case Version:
		return response.Message != "" && response.MaxChannel == 0
	case MaxChannels:
		return response.MaxChannel > 0
	}
	return true
}

// ParseAnswer parses responses from the reader until it finds one that
// answers the command, skipping responses to earlier commands.
func (response *Response) ParseAnswer(reader *bufio.Reader, command Command) error {
	for {
		*response = Response{}
		if err := response.Parse(reader); err != nil {
			return err
		}
		if response.Answers(command) {
			return nil
		}
	}
}
