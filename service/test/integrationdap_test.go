package service_test

import (
	"bytes"
	"net"
	"testing"
	"time"

	protest "github.com/go-delve/delve/pkg/proc/test"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/dap"
)

func startDAPServer(t *testing.T) (server *dap.Server, addr string) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	server = dap.NewServer(&service.Config{
		Listener:       listener,
		Backend:        "default",
		DisconnectChan: nil,
	})
	server.Run()
	// Give server time to start listening for clients
	time.Sleep(100 * time.Millisecond)
	return server, listener.Addr().String()
}

func expectMessage(t *testing.T, client *DAPClient, want []byte) {
	got, err := client.ReadBaseMessage()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("\ngot  %q\nwant %q", got, want)
	}
}

// name is for _fixtures/<name>.go
func runTest(t *testing.T, name string, test func(c *DAPClient, f protest.Fixture)) {
	var buildFlags protest.BuildFlags
	fixture := protest.BuildFixture("helloworld", buildFlags)

	server, addr := startDAPServer(t)
	client := NewDAPClient(addr)
	defer client.Close()
	defer server.Stop()

	test(client, fixture)
}

func TestStopOnEntry(t *testing.T) {
	runTest(t, "helloworld", func(client *DAPClient, fixture protest.Fixture) {
		client.InitializeRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":0,"success":true,"command":"initialize","body":{"supportsConfigurationDoneRequest":true}}`))

		client.LaunchRequest(fixture.Path, true /*stopOnEntry*/)
		expectMessage(t, client, []byte(`{"seq":0,"type":"event","event":"initialized"}`))
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":1,"success":true,"command":"launch"}`))

		client.SetExceptionBreakpointsRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":2,"success":true,"command":"setExceptionBreakpoints"}`))

		client.ConfigurationDoneRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"event","event":"stopped","body":{"reason":"breakpoint","threadId":1,"allThreadsStopped":true}}`))
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":3,"success":true,"command":"configurationDone"}`))

		client.ContinueRequest(1)
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":4,"success":true,"command":"continue","body":{}}`))
		expectMessage(t, client, []byte(`{"seq":0,"type":"event","event":"terminated","body":{}}`))

		client.DisconnectRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":5,"success":true,"command":"disconnect"}`))
	})
}

func TestSetBreakpoint(t *testing.T) {
	runTest(t, "helloworld", func(client *DAPClient, fixture protest.Fixture) {
		client.InitializeRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":0,"success":true,"command":"initialize","body":{"supportsConfigurationDoneRequest":true}}`))

		client.LaunchRequest(fixture.Path, false /*stopOnEntry*/)
		expectMessage(t, client, []byte(`{"seq":0,"type":"event","event":"initialized"}`))
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":1,"success":true,"command":"launch"}`))

		client.SetBreakpointsRequest(fixture.Source, []int{8})
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":2,"success":true,"command":"setBreakpoints","body":{"breakpoints":[{"verified":true,"source":{},"line":8}]}}`))

		client.SetExceptionBreakpointsRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":3,"success":true,"command":"setExceptionBreakpoints"}`))

		client.ConfigurationDoneRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":4,"success":true,"command":"configurationDone"}`))

		client.ContinueRequest(1)
		expectMessage(t, client, []byte(`{"seq":0,"type":"event","event":"stopped","body":{"reason":"breakpoint","threadId":1,"allThreadsStopped":true}}`))
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":5,"success":true,"command":"continue","body":{}}`))

		client.ContinueRequest(1)
		expectMessage(t, client, []byte(`{"seq":0,"type":"event","event":"terminated","body":{}}`))
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":6,"success":true,"command":"continue","body":{}}`))

		client.DisconnectRequest()
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":7,"success":true,"command":"disconnect"}`))
	})
}

func TestBadLaunchRequests(t *testing.T) {
	runTest(t, "helloworld", func(client *DAPClient, fixture protest.Fixture) {
		client.LaunchRequest("", true)
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":0,"success":false,"command":"launch","message":"Failed to launch","body":{"error":{"id":3000,"format":"Failed to launch: The program attribute is missing in debug configuration."}}}`))

		client.LaunchRequest(fixture.Path+"bad", false)
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":1,"success":false,"command":"launch","message":"Failed to launch","body":{"error":{"id":3000,"format":"Failed to launch: could not launch process: stub exited while waiting for connection: exit status 0"}}}`))

		client.LaunchRequest(fixture.Source, true)
		expectMessage(t, client, []byte(`{"seq":0,"type":"response","request_seq":2,"success":false,"command":"launch","message":"Failed to launch","body":{"error":{"id":3000,"format":"Failed to launch: not an executable file"}}}`))
	})
}
