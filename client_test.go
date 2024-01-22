package esl

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	addr, password := os.Getenv("ESL_ADDR"), os.Getenv("ESL_PASSWORD")
	if addr == "" {
		addr = "localhost"
	}

	events := make(chan Event, 1)
	go func() {
		for ev := range events {
			t.Logf("event %s [%d]", ev.Name(), ev.Sequence())
			if ev.ContentLength() > 0 {
				t.Logf("body: %s", ev.Body())
			}
		}
		t.Log("events channel closed")
	}()

	client, err := Connect(addr, password,
		WithEvents(events, true),
		WithLog(slog.Default()),
		// WithDumpIn(os.Stdout),
		// WithDumpOut(os.Stderr),
	)
	if err != nil {
		t.Skip("FreeSWITCH not running:", err)
	}
	defer client.Close()

	if err = client.Subscribe("all"); err != nil {
		t.Error(err)
	}

	msg, err := client.API("status")
	if err != nil {
		t.Error(err)
	}
	_ = msg
	// t.Log(msg)

	// spell-checker:ignore msleep
	err = client.JobWithID("msleep 3000", "test")
	if err != nil {
		t.Error(err)
	}

	jobid, err := client.Job("msleep 2000")
	if err != nil {
		t.Error(err)
	}
	if jobid == "" {
		t.Error("incorrect job id:", jobid)
	}

	time.Sleep(time.Second * 5)
}

// for _, cmd := range []string{
// 	// spell-checker:disable
// 	"fsctl debug_level 9",
// 	"fsctl loglevel 7",
// 	"fsctl debug_sql",
// 	"fsctl last_sps",
// 	`json {"command" : "status", "data" : ""}`,
// 	"md5 freeswitch-is-awesome",
// 	"module_exists mod_callcenter",
// 	"show channels as json",
// 	"status",
// 	// spell-checker:enable
// } {
// 	_, err := client.Job(cmd)
// 	if err != nil {
// 		return err
// 	}
// }

func TestClientDefault(t *testing.T) {
	client, err := Connect("", "ClueCon",
		WithLog(slog.Default()),
	)
	if err != nil {
		t.Skip("FreeSWITCH not running:", err)
	}
	client.Close()
}
