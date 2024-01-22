package esl

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestConnection_Read(t *testing.T) {
	f, err := os.Open("test_data.log")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := newConn(f, nil)
	for {
		resp, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("resp:  %s [len: %d]", resp.ContentType(), resp.ContentLength())
		if resp.isZero() {
			t.Error("response is empty")
		}
		_ = resp.String()

		if resp.ContentType() != "text/event-plain" {
			continue
		}

		event, err := resp.toEvent()
		if err != nil {
			t.Error(err)
			continue
		}

		if len(event.headers) == 0 {
			t.Error("event is empty")
		}

		t.Logf("event: %s (seq: %d)", event.Name(), event.Sequence())
		if event.ContentLength() > 0 {
			t.Logf("body:\n%s", event.Body())
		}
	}
}
