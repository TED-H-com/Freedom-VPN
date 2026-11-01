package members

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

// The aggregate stream grows through the assembled run. Bound each record,
// never the whole export, and reject any partial trailing append.
func parseTimingStream(reader io.Reader) ([]json.RawMessage, error) {
	rows := []json.RawMessage{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		raw := scanner.Bytes()
		if !json.Valid(raw) {
			return nil, fmt.Errorf("incomplete producer record")
		}
		if err := validateProducerCommands(raw); err != nil {
			return nil, err
		}
		rows = append(rows, append(json.RawMessage(nil), raw...))
	}
	if e := scanner.Err(); e != nil {
		return nil, e
	}
	return rows, nil
}

func TestAT02_F12TimingStreamExceedsDiagnosticLimit(t *testing.T) {
	row := []byte(`{"source":"TEST ONLY","sequence":"9007199254740993"}` + "\n")
	raw := bytes.Repeat(row, 40000)
	if len(raw) <= 1<<20 {
		t.Fatal("fixture too small")
	}
	rows, e := parseTimingStream(bytes.NewReader(raw))
	if e != nil || len(rows) != 40000 || !bytes.Equal(rows[0], bytes.TrimSpace(row)) {
		t.Fatal("large stream lost", len(rows), e)
	}
	if _, e = parseTimingStream(bytes.NewReader(append(raw, []byte(`{"source":`)...))); e == nil {
		t.Fatal("partial tail accepted")
	}
}
