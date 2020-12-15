package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

	bodyReader = bytes.NewReader(response.body)

	cmd = exec.Command(proc.Cmd.Name)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Stdin = bodyReader
	cmd.Args = append(cmd.Args, proc.Cmd.Args...)

	err := cmd.Run()
	if err != nil {
		var (
			err2        error
			exitCode    int
			stderrBytes []byte
		)

		stderrBytes = stderr.Bytes()

		exitError, ok := err.(*exec.ExitError)
		if ok {
			exitCode = exitError.ExitCode()
			stderrBytes = append(stderrBytes, exitError.Stderr...)
		}

		switch proc.Cmd.Output {
		case CmdOutputExitCode:
			response.body = []byte(strconv.Itoa(exitCode))
		case CmdOutputStderr:
			response.body = stderrBytes
		case CmdOutputStdout:
			response.body, err2 = json.MarshalIndent(preProcessError{
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
		response.body = []byte(strconv.Itoa(cmd.ProcessState.ExitCode()))
	case CmdOutputStderr:
		response.body = stderr.Bytes()
	case CmdOutputStdout:
		response.body = stdout.Bytes()
	}

	return response, nil
}
