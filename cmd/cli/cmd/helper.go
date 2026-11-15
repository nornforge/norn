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
	timeout, err := rootCmd.PersistentFlags().GetDuration("timeout")
	if err != nil {
		return response, err
	}
	// Data still waiting in the port, such as the late response to an
	// earlier command, would be taken as the response to this one.
	if err := port.ResetInputBuffer(); err != nil {
		return response, fmt.Errorf("error while clearing the input of %s: %w", portName, err)
	}
	if _, err := port.Write(command.Marshal()); err != nil {
		return response, fmt.Errorf("error while sending the command to %s: %w", portName, err)
	}
	reader := bufio.NewReader(newTimeoutReader(port, timeout))
	err = response.Parse(reader)
	if err != nil {
		return response, err
	}
	if !response.Success {
		return response, fmt.Errorf("%s", response.Message)
	}
	return response, nil
}
