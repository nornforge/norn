package cmd

import (
	"bufio"
	"fmt"
	"log"

	"github.com/nornforge/norn/pkg/norn"
	"go.bug.st/serial"
)

func sendCommandOverSerial(command norn.Command) (norn.Response, error) {
	log.SetFlags(0)
	response := norn.Response{
		Success: false,
	}
	portName, err := rootCmd.PersistentFlags().GetString("device")
	if err != nil {
		return response, err
	}
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		log.Printf("Error: while opening port: %s", portName)
		return response, err
	}
	defer port.Close()
	port.Write(command.Marshal())
	reader := bufio.NewReader(port)
	err = response.Parse(reader)
	if err != nil {
		return response, err
	}
	if !response.Success {
		return response, fmt.Errorf(response.Message)
	}
	return response, nil
}
