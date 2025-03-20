package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/programmfabrik/golib"
)

type CmdOutputType string

const (
	CmdOutputStdout   CmdOutputType = "stdout"
	CmdOutputStderr   CmdOutputType = "stderr"
	CmdOutputExitCode CmdOutputType = "exitcode"
)

type PreProcess struct {
	Cmd struct {
		Name   string        `json:"name"`
		Args   []string      `json:"args,omitempty"`
		Output CmdOutputType `json:"output,omitempty"`
	} `json:"cmd"`
}

type preProcessError struct {
	Command  string `json:"command"`
	Error    string `json:"error"`
	ExitCode int    `json:"exit_code"`
	StdErr   string `json:"stderr"`
}

func (proc *PreProcess) RunPreProcess(response Response) (Response, error) {

	var (
		stdout     bytes.Buffer
		stderr     bytes.Buffer
		bodyReader *bytes.Reader
		cmd        *exec.Cmd
	)

	if proc.Cmd.Name == "" {
		return response, fmt.Errorf("Invalid cmd for pre_process: must not be an empty string")
	}

	if proc.Cmd.Output == "" {
		proc.Cmd.Output = CmdOutputStdout
	}

	bodyReader = bytes.NewReader(response.Body)

	cmd = exec.Command(proc.Cmd.Name)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Stdin = bodyReader
	cmd.Args = append(cmd.Args, proc.Cmd.Args...)

	err := cmd.Run()
	if err != nil {
		stderrBytes := stderr.Bytes()
		var exitCode int
		var err2 error
		exitError, ok := err.(*exec.ExitError)
		if ok {
			exitCode = exitError.ExitCode()
			stderrBytes = append(stderrBytes, exitError.Stderr...)
		}

		switch proc.Cmd.Output {
		case CmdOutputExitCode:
			response.Body = []byte(strconv.Itoa(exitCode))
		case CmdOutputStderr:
			response.Body = ensureJson(stderrBytes)
		case CmdOutputStdout:
			response.Body, err2 = golib.JsonBytesIndent(preProcessError{
				Command:  strings.Join(cmd.Args, " "),
				Error:    err.Error(),
				ExitCode: exitCode,
				StdErr:   string(stderrBytes),
			}, "", "  ")
		}

		return response, err2
	}

	switch proc.Cmd.Output {
	case CmdOutputExitCode:
		response.Body = []byte(strconv.Itoa(cmd.ProcessState.ExitCode()))
	case CmdOutputStderr:
		response.Body = ensureJson(stderr.Bytes())
	case CmdOutputStdout:
		response.Body = ensureJson(stdout.Bytes())
	}

	return response, nil
}

// ensureJson makes sure that data can be parsed as JSON. In case
// that data is something like '0 (0)' the data is wrapped as string.
func ensureJson(data []byte) (dataFixed []byte) {
	var v any
	parseErr := json.Unmarshal(data, &v)
	if parseErr != nil {
		dataFixed, _ = golib.JsonBytes(string(data))
		return dataFixed
	}
	return data
}
