package streamstats

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
)

func TestF802_PartialLineAndPayloadFreeCounts(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go func() { _, _ = io.WriteString(server, "TEST ONLY payload\npartial"); _ = server.Close() }()
	c := Wrap(client)
	reader := bufio.NewReader(c)
	line, e := reader.ReadString('\n')
	c.Line(len(line), e == nil, e)
	line, e = reader.ReadString('\n')
	c.Line(len(line), e == nil, e)
	got := c.Snapshot()
	if got.ReadBytes != 25 || got.Lines != 1 || got.Partial != 7 || got.Error != "eof" {
		t.Fatalf("counts: %+v", got)
	}
	b, e := json.Marshal(got)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), "TEST ONLY payload") || strings.Contains(string(b), "partial\"") {
		t.Fatal("payload leaked")
	}
}
