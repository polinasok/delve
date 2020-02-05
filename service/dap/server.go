package dap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"

	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/google/go-dap"
	"github.com/sirupsen/logrus"
)

// Server implements a DAP server that can accept a single client for a single
// debug session. It does not support restarting.
type Server struct {
	// config is all the information necessary to start the debugger and server.
	config *service.Config
	// listener is used to serve HTTP.
	listener net.Listener
	// conn is accepted client connection.
	conn net.Conn
	// reader is used to read requests.
	reader *bufio.Reader
	// debugger is the debugger service.
	debugger *debugger.Debugger
	// log is used for structured logging.
	log *logrus.Entry
	// stopOnEntry is set to automatically stop program after start
	stopOnEntry bool
}

// NewServer creates a new DAP Server.
func NewServer(config *service.Config) *Server {
	logger := logflags.DAPLogger()
	logflags.WriteDAPListeningMessage(config.Listener.Addr().String())
	return &Server{
		config:   config,
		listener: config.Listener,
		log:      logger,
	}
}

// Stop stops the DAP debugger service.
// It kills the target process if it was launched by the debugger.
func (s *Server) Stop() error {
	s.listener.Close()
	if s.debugger != nil {
		kill := s.config.AttachPid == 0
		return s.debugger.Detach(kill)
	}
	return nil
}

// Run starts a debugger and exposes it with an HTTP server. The debugger
// itself can be stopped with the `detach` API. Run blocks until the HTTP
// server stops.
func (s *Server) Run() {
	go func() {
		// For now, do not support multiple clients in dap mode, serially or in parallel.
		// The server should be restarted for every new debug session.
		// TODO(polina): allow new client connections for new debug sessions?
		defer s.listener.Close()
		conn, err := s.listener.Accept()
		if err != nil {
			panic(err)
		}
		s.conn = conn
		go s.serveDAPCodec()
	}()
}

func (s *Server) serveDAPCodec() {
	defer func() {
		if s.config.DisconnectChan != nil {
			close(s.config.DisconnectChan)
		}
		s.conn.Close()
	}()
	s.reader = bufio.NewReader(s.conn)
	for {
		request, err := dap.ReadProtocolMessage(s.reader)
		// TODO(polina): Differentiate between connection and decoding
		// errors. If we get an unfamiliar request, should we respond
		// with a no-op or an ErrorResponse?
		if err != nil {
			if err != io.EOF {
				s.log.Error("DAP error:", err)
			}
			break
		}
		s.handleRequest(request)
	}
}

func (s *Server) handleRequest(request dap.Message) {
	jsonmsg, _ := json.Marshal(request)
	s.log.Debug("[<- from client]", string(jsonmsg))

	switch request := request.(type) {
	case *dap.InitializeRequest:
		s.onInitializeRequest(request)
	case *dap.LaunchRequest:
		s.onLaunchRequest(request)
	case *dap.AttachRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onAttachRequest(request)
	case *dap.DisconnectRequest:
		s.onDisconnectRequest(request)
	case *dap.TerminateRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onTerminateRequest(request)
	case *dap.RestartRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onRestartRequest(request)
	case *dap.SetBreakpointsRequest:
		s.onSetBreakpointsRequest(request)
	case *dap.SetFunctionBreakpointsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onSetFunctionBreakpointsRequest(request)
	case *dap.SetExceptionBreakpointsRequest:
		s.onSetExceptionBreakpointsRequest(request)
	case *dap.ConfigurationDoneRequest:
		s.onConfigurationDoneRequest(request)
	case *dap.ContinueRequest:
		s.onContinueRequest(request)
	case *dap.NextRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onNextRequest(request)
	case *dap.StepInRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onStepInRequest(request)
	case *dap.StepOutRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onStepOutRequest(request)
	case *dap.StepBackRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onStepBackRequest(request)
	case *dap.ReverseContinueRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onReverseContinueRequest(request)
	case *dap.RestartFrameRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onRestartFrameRequest(request)
	case *dap.GotoRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onGotoRequest(request)
	case *dap.PauseRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onPauseRequest(request)
	case *dap.StackTraceRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onStackTraceRequest(request)
	case *dap.ScopesRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onScopesRequest(request)
	case *dap.VariablesRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onVariablesRequest(request)
	case *dap.SetVariableRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onSetVariableRequest(request)
	case *dap.SetExpressionRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onSetExpressionRequest(request)
	case *dap.SourceRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onSourceRequest(request)
	case *dap.ThreadsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onThreadsRequest(request)
	case *dap.TerminateThreadsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onTerminateThreadsRequest(request)
	case *dap.EvaluateRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onEvaluateRequest(request)
	case *dap.StepInTargetsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onStepInTargetsRequest(request)
	case *dap.GotoTargetsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onGotoTargetsRequest(request)
	case *dap.CompletionsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onCompletionsRequest(request)
	case *dap.ExceptionInfoRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onExceptionInfoRequest(request)
	case *dap.LoadedSourcesRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onLoadedSourcesRequest(request)
	case *dap.DataBreakpointInfoRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onDataBreakpointInfoRequest(request)
	case *dap.SetDataBreakpointsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onSetDataBreakpointsRequest(request)
	case *dap.ReadMemoryRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onReadMemoryRequest(request)
	case *dap.DisassembleRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onDisassembleRequest(request)
	case *dap.CancelRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onCancelRequest(request)
	case *dap.BreakpointLocationsRequest:
		s.sendUnsupportedErrorResponse(request.Request)
		//s.onBreakpointLocationsRequest(request)
	default:
		// This should have been returned as an error from
		// dap.ReadProtocolMessage(), so we should never reach here.
		panic(fmt.Sprintf("Unable to process %#v", request))
	}
}

func (s *Server) send(message dap.Message) {
	jsonmsg, _ := json.Marshal(message)
	s.log.Debug("[-> to client]", string(jsonmsg))
	dap.WriteProtocolMessage(s.conn, message)
}

func (s *Server) onInitializeRequest(request *dap.InitializeRequest) {
	// TODO(polina): Respond with an error if debug session is in progress?
	response := &dap.InitializeResponse{Response: *newResponse(request.Request)}
	response.Body.SupportsConfigurationDoneRequest = true
	// TODO(polina): support this to match vscode-go functionality
	response.Body.SupportsSetVariable = false
	s.send(response)
}

func (s *Server) onLaunchRequest(request *dap.LaunchRequest) {
	// TODO(polina): Respond with an error if debug session is in progress?
	program, ok := request.Arguments["program"]
	if !ok || program == "" {
		s.sendErrorResponse(request.Request,
			3000, "Failed to launch",
			"The program attribute is missing in debug configuration.")
		return
	}
	s.config.ProcessArgs = []string{program.(string)}
	s.config.WorkingDir = filepath.Dir(program.(string))
	// TODO: support program args

	stop, ok := request.Arguments["stopOnEntry"]
	s.stopOnEntry = (ok && stop == true)

	mode, ok := request.Arguments["mode"]
	if !ok || mode == "" {
		mode = "debug"
	}
	// TODO(polina): support "debug", "test" and "remote" modes
	if mode != "exec" {
		s.sendErrorResponse(request.Request,
			3000, "Failed to launch",
			fmt.Sprintf("Unsupported 'mode' value '%s' in debug configuration.", mode))
		return
	}

	config := &debugger.Config{
		WorkingDir:           s.config.WorkingDir,
		AttachPid:            0,
		CoreFile:             "",
		Backend:              s.config.Backend,
		Foreground:           s.config.Foreground,
		DebugInfoDirectories: s.config.DebugInfoDirectories,
		CheckGoVersion:       s.config.CheckGoVersion,
	}
	var err error
	if s.debugger, err = debugger.New(config, s.config.ProcessArgs); err != nil {
		s.sendErrorResponse(request.Request,
			3000, "Failed to launch", err.Error())
		return
	}

	// Notify the client that the debugger is ready to start "accepting"
	// configuration requests for setting breakpoints, etc. The client
	// will end the configuration sequence with 'configurationDone'.
	s.send(&dap.InitializedEvent{Event: *newEvent("initialized")})
	s.send(&dap.LaunchResponse{Response: *newResponse(request.Request)})
}

func (s *Server) onDisconnectRequest(request *dap.DisconnectRequest) {
	s.send(&dap.DisconnectResponse{Response: *newResponse(request.Request)})
	// TODO(polina): only halt if the program is running
	_, err := s.debugger.Command(&api.DebuggerCommand{Name: api.Halt})
	if err != nil {
		s.log.Error(err)
	}
	kill := s.config.AttachPid == 0
	err = s.debugger.Detach(kill)
	if err != nil {
		s.log.Error(err)
	}
	if s.config.DisconnectChan != nil {
		close(s.config.DisconnectChan)
		s.config.DisconnectChan = nil
	}
}

func (s *Server) onSetBreakpointsRequest(request *dap.SetBreakpointsRequest) {
	if request.Arguments.Source.Path == "" {
		s.log.Error("ERROR: Unable to set breakpoint for empty file path")
	}
	response := &dap.SetBreakpointsResponse{Response: *newResponse(request.Request)}
	response.Body.Breakpoints = make([]dap.Breakpoint, len(request.Arguments.Breakpoints))
	i := 0
	for _, b := range request.Arguments.Breakpoints {
		bp, err := s.debugger.CreateBreakpoint(
			&api.Breakpoint{File: request.Arguments.Source.Path, Line: b.Line})
		if err != nil {
			s.log.Error("ERROR:", err)
			continue
		}
		response.Body.Breakpoints[i].Verified = true
		response.Body.Breakpoints[i].Line = bp.Line
		i++
	}
	response.Body.Breakpoints = response.Body.Breakpoints[:i]
	s.send(response)
}

func (s *Server) onSetExceptionBreakpointsRequest(request *dap.SetExceptionBreakpointsRequest) {
	// This request is always sent although we specify empty filters at initialization.
	// Handle it as a no-op.
	s.send(&dap.SetExceptionBreakpointsResponse{Response: *newResponse(request.Request)})
}

func (s *Server) onConfigurationDoneRequest(request *dap.ConfigurationDoneRequest) {
	if s.stopOnEntry {
		e := &dap.StoppedEvent{
			Event: *newEvent("stopped"),
			Body:  dap.StoppedEventBody{Reason: "breakpoint", ThreadId: 1, AllThreadsStopped: true},
		}
		s.send(e)
	}
	s.send(&dap.ConfigurationDoneResponse{Response: *newResponse(request.Request)})
	if !s.stopOnEntry {
		s.doContinue()
	}
}

func (s *Server) onContinueRequest(request *dap.ContinueRequest) {
	s.send(&dap.ContinueResponse{Response: *newResponse(request.Request)})
	s.doContinue()
}

func (s *Server) sendErrorResponse(request dap.Request, id int, summary string, details string) {
	er := &dap.ErrorResponse{}
	er.Type = "response"
	er.Command = request.Command
	er.RequestSeq = request.Seq
	er.Success = false
	er.Message = summary
	er.Body.Error.Id = id
	er.Body.Error.Format = fmt.Sprintf("%s: %s", summary, details)
	s.log.Error(er.Body.Error.Format)
	s.send(er)
}

func (s *Server) sendUnsupportedErrorResponse(request dap.Request) {
	s.sendErrorResponse(request, 9999, "Unsupported command",
		fmt.Sprintf("cannot process '%s' request", request.Command))
}

func newResponse(request dap.Request) *dap.Response {
	return &dap.Response{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  0,
			Type: "response",
		},
		Command:    request.Command,
		RequestSeq: request.Seq,
		Success:    true,
	}
}

func newEvent(event string) *dap.Event {
	return &dap.Event{
		ProtocolMessage: dap.ProtocolMessage{
			Seq:  0,
			Type: "event",
		},
		Event: event,
	}
}

func (s *Server) doContinue() {
	if s.debugger == nil {
		return
	}
	state, err := s.debugger.Command(&api.DebuggerCommand{Name: api.Continue})
	if err != nil {
		s.log.Error(err)
		switch err.(type) {
		case proc.ErrProcessExited:
			e := &dap.TerminatedEvent{Event: *newEvent("terminated")}
			s.send(e)
		default:
		}
		return
	}
	if state.Exited {
		e := &dap.TerminatedEvent{Event: *newEvent("terminated")}
		s.send(e)
	} else {
		e := &dap.StoppedEvent{Event: *newEvent("stopped")}
		e.Body.Reason = "breakpoint"
		e.Body.AllThreadsStopped = true
		e.Body.ThreadId = state.SelectedGoroutine.ID
		s.send(e)
	}
}
