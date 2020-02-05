package service_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"path/filepath"

	"github.com/google/go-dap"
)

// DAPClient is a debugger service client that uses Debug Adaptor Protocol.
// It does not (yet?) implement service.Client interface.
// All client methods are synchronous.
type DAPClient struct {
	conn   net.Conn
	reader *bufio.Reader
	// seq is used to track the sequence number of each
	// requests that the client sends to the server
	seq int
}

// Close closes the client connection
func (c *DAPClient) Close() {
	c.conn.Close()
}

// NewDAPClient creates a new DAPClient over a TCP connection.
func NewDAPClient(addr string) *DAPClient {
	fmt.Println("Connecting to server at:", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	c := &DAPClient{conn: conn, reader: bufio.NewReader(conn)}
	return c
}

func (c *DAPClient) send(request dap.Message) {
	jsonmsg, _ := json.Marshal(request)
	fmt.Println("[->   to server]", string(jsonmsg))
	dap.WriteProtocolMessage(c.conn, request)
}

// ReadBaseMessage reads and returns a json-encoded DAP message.
func (c *DAPClient) ReadBaseMessage() ([]byte, error) {
	message, err := dap.ReadBaseMessage(c.reader)
	if err != nil {
		fmt.Println("DAP client error:", err)
		return nil, err
	}
	fmt.Println("[<- from server]", string(message))
	return message, nil
}

// InitializeRequest sends an 'initialize' request.
func (c *DAPClient) InitializeRequest() {
	request := &dap.InitializeRequest{Request: *c.newRequest("initialize")}
	request.Arguments = dap.InitializeRequestArguments{
		AdapterID:                    "go",
		PathFormat:                   "path",
		LinesStartAt1:                true,
		ColumnsStartAt1:              true,
		SupportsVariableType:         true,
		SupportsVariablePaging:       true,
		SupportsRunInTerminalRequest: true,
		Locale:                       "en-us",
	}
	c.send(request)
}

// LaunchRequest sends a 'launch' request.
func (c *DAPClient) LaunchRequest(program string, stopOnEntry bool) {
	request := &dap.LaunchRequest{Request: *c.newRequest("launch")}
	request.Arguments = map[string]interface{}{
		"request":     "launch",
		"mode":        "exec",
		"program":     program,
		"stopOnEntry": stopOnEntry,
	}
	c.send(request)
}

// DisconnectRequest sends a 'disconnect' request.
func (c *DAPClient) DisconnectRequest() {
	request := &dap.DisconnectRequest{Request: *c.newRequest("disconnect")}
	c.send(request)
}

// SetBreakpointsRequest sends a 'setBreakpoints' request.
func (c *DAPClient) SetBreakpointsRequest(file string, lines []int) {
	request := &dap.SetBreakpointsRequest{Request: *c.newRequest("setBreakpoints")}
	request.Arguments = dap.SetBreakpointsArguments{
		Source: dap.Source{
			Name: filepath.Base(file),
			Path: file,
		},
		Breakpoints: make([]dap.SourceBreakpoint, len(lines)),
		//sourceModified: false,
	}
	for i, l := range lines {
		request.Arguments.Breakpoints[i].Line = l
	}
	c.send(request)
}

// SetExceptionBreakpointsRequest sends a 'setExceptionBreakpoints' request.
func (c *DAPClient) SetExceptionBreakpointsRequest() {
	request := &dap.SetBreakpointsRequest{Request: *c.newRequest("setExceptionBreakpoints")}
	c.send(request)
}

// ConfigurationDoneRequest sends a 'configurationDone' request.
func (c *DAPClient) ConfigurationDoneRequest() {
	request := &dap.ConfigurationDoneRequest{Request: *c.newRequest("configurationDone")}
	c.send(request)
}

// ContinueRequest sends a 'continue' request.
func (c *DAPClient) ContinueRequest(thread int) {
	request := &dap.ContinueRequest{Request: *c.newRequest("continue")}
	request.Arguments.ThreadId = thread
	c.send(request)
}

func (c *DAPClient) newRequest(command string) *dap.Request {
	request := &dap.Request{}
	request.Type = "request"
	request.Command = command
	request.Seq = c.seq
	c.seq++
	return request
}
