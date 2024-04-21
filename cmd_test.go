package esl

import (
	"log/slog"
	"testing"
)

func TestCmd(t *testing.T) {
	tests := []struct {
		command
		want string
	}{
		// spell-checker:disable
		{
			cmd("api", "show calls count"),
			"api show calls count",
		},
		{
			cmd("api", "uptime", "s"),
			"api uptime s",
		},
		{
			cmd("api", "strepoch"),
			"api strepoch",
		},
		{
			cmd("api", "status"),
			"api status",
		},
		{
			cmd("api", "originate sofia/mydomain.com/ext@yourvsp.com 1000"),
			"api originate sofia/mydomain.com/ext@yourvsp.com 1000",
		},
		{
			cmd("bgapi", "status"),
			"bgapi status",
		},
		{
			cmd("bgapi", "status").WithJobUUID("d29a070f-40ff-43d8-8b9d-d369b2389dfe"),
			"bgapi status\nJob-UUID: d29a070f-40ff-43d8-8b9d-d369b2389dfe",
		},
		{
			cmd("linger"),
			"linger",
		},
		{
			cmd("nolinger"),
			"nolinger",
		},
		{
			cmd("event", "plain ALL"),
			"event plain ALL",
		},
		{
			cmd("event", "plain", "CHANNEL_CREATE", "CHANNEL_DESTROY", "CUSTOM", "conference::maintenance",
				"sofia::register", "sofia::expire"),
			"event plain CHANNEL_CREATE CHANNEL_DESTROY CUSTOM conference::maintenance sofia::register sofia::expire",
		},
		{
			cmd("myevents", "d29a070f-40ff-43d8-8b9d-d369b2389dfe"),
			"myevents d29a070f-40ff-43d8-8b9d-d369b2389dfe",
		},
		{
			cmd("divert_events", "on"),
			"divert_events on",
		},
		{
			cmd("filter", "Event-Name", "CHANNEL_EXECUTE"),
			"filter Event-Name CHANNEL_EXECUTE",
		},
		{
			cmd("filter", "Unique-ID", "d29a070f-40ff-43d8-8b9d-d369b2389dfe"),
			"filter Unique-ID d29a070f-40ff-43d8-8b9d-d369b2389dfe",
		},
		{
			cmd("filter delete", "Unique-ID", "d29a070f-40ff-43d8-8b9d-d369b2389dfe"),
			"filter delete Unique-ID d29a070f-40ff-43d8-8b9d-d369b2389dfe",
		},
		{
			cmd("filter delete", "Unique-ID"),
			"filter delete Unique-ID",
		},
		{
			cmd("sendevent", "SEND_INFO").
				WithMessage(map[string]string{
					"profile":      "external",
					"content-type": "text/plain",
					"to-uri":       "sip:1@2.3.4.5",
					"from-uri":     "sip:1@1.2.3.4",
				}, "test"),
			"sendevent SEND_INFO\n" +
				"content-type: text/plain\n" +
				"from-uri: sip:1@1.2.3.4\n" +
				"profile: external\n" +
				"to-uri: sip:1@2.3.4.5\n" +
				"content-length: 4\n" +
				"\n" +
				"test",
		},
		{
			cmd("sendevent", "message_waiting").WithMessage(map[string]string{
				"MWI-Messages-Waiting": "no",
				"MWI-Message-Account":  "sip:user1@192.168.1.14",
			}, ""),
			"sendevent message_waiting\n" +
				"MWI-Message-Account: sip:user1@192.168.1.14\n" +
				"MWI-Messages-Waiting: no",
		},
		{
			cmd("sendevent", "message_waiting").WithMessage(map[string]string{
				"MWI-Messages-Waiting": "yes",
				"MWI-Message-Account":  "sip:user1@192.168.1.14",
				"MWI-Voice-Message":    "0/1 (0/0)",
			}, ""),
			"sendevent message_waiting\n" +
				"MWI-Message-Account: sip:user1@192.168.1.14\n" +
				"MWI-Messages-Waiting: yes\n" +
				"MWI-Voice-Message: 0/1 (0/0)",
		},
		{
			cmd("sendevent", "NOTIFY").WithMessage(map[string]string{
				"profile":      "internal",
				"event-string": "check-sync;reboot=false",
				"user":         "1000",
				"host":         "192.168.10.4",
				"content-type": "application/simple-message-summary",
			}, ""),
			"sendevent NOTIFY\n" +
				"content-type: application/simple-message-summary\n" +
				"event-string: check-sync;reboot=false\n" +
				"host: 192.168.10.4\n" +
				"profile: internal\n" +
				"user: 1000",
		},
		{
			cmd("sendevent", "NOTIFY").WithMessage(map[string]string{
				"profile":      "internal",
				"event-string": "check-sync",
				"user":         "1005",
				"host":         "192.168.10.4",
				"content-type": "application/simple-message-summary",
			}, "OK"),
			"sendevent NOTIFY\n" +
				"content-type: application/simple-message-summary\n" +
				"event-string: check-sync\n" +
				"host: 192.168.10.4\n" +
				"profile: internal\n" +
				"user: 1005\n" +
				"content-length: 2\n" +
				"\n" +
				"OK",
		},
		{
			cmd("sendevent", "SEND_MESSAGE").WithMessage(map[string]string{
				"profile":      "internal",
				"content-type": "text/plain",
				"user":         "1005",
				"host":         "99.157.44.194",
			}, "OK"),
			"sendevent SEND_MESSAGE\n" +
				"content-type: text/plain\n" +
				"host: 99.157.44.194\n" +
				"profile: internal\n" +
				"user: 1005\n" +
				"content-length: 2\n" +
				"\n" +
				"OK",
		},
		{
			cmd("sendevent", "NOTIFY").WithMessage(map[string]string{
				"profile":      "internal",
				"event-string": "check-sync",
				"user":         "1005",
				"host":         "99.157.44.194",
				"content-type": "application/simple-message-summary",
			}, "OK"),
			"sendevent NOTIFY\n" +
				"content-type: application/simple-message-summary\n" +
				"event-string: check-sync\n" +
				"host: 99.157.44.194\n" +
				"profile: internal\n" +
				"user: 1005\n" +
				"content-length: 2\n" +
				"\n" +
				"OK",
		},
		{
			cmd("sendevent", "NOTIFY").WithMessage(map[string]string{
				"profile":      "internal",
				"event-string": "resync;profile=http://10.20.30.40/profile.xml",
				"user":         "1000",
				"host":         "10.20.30.40",
				"content-type": "application/simple-message-summary",
				"to-uri":       "sip:1000@10.20.30.40",
				"from-uri":     "sip:1000@10.20.30.40",
			}, ""),
			"sendevent NOTIFY\n" +
				"content-type: application/simple-message-summary\n" +
				"event-string: resync;profile=http://10.20.30.40/profile.xml\n" +
				"from-uri: sip:1000@10.20.30.40\n" +
				"host: 10.20.30.40\n" +
				"profile: internal\n" +
				"to-uri: sip:1000@10.20.30.40\n" +
				"user: 1000",
		},
		{
			cmd("sendevent", "SWITCH_EVENT_PHONE_FEATURE").WithMessage(map[string]string{
				"profile":        "internal",
				"user":           "ex1004",
				"host":           "3.local",
				"device":         "ex1004",
				"Feature-Event":  "DoNotDisturbEvent",
				"doNotDisturbOn": "on",
			}, ""),
			"sendevent SWITCH_EVENT_PHONE_FEATURE\n" +
				"Feature-Event: DoNotDisturbEvent\n" +
				"device: ex1004\n" +
				"doNotDisturbOn: on\n" +
				"host: 3.local\n" +
				"profile: internal\n" +
				"user: ex1004",
		},
		{
			cmd("sendmsg", "<uuid>").WithMessage(map[string]string{
				"call-command":     "execute",
				"execute-app-name": "playback",
				"execute-app-arg":  "/tmp/test.wav",
			}, ""),
			"sendmsg <uuid>\n" +
				"call-command: execute\n" +
				"execute-app-arg: /tmp/test.wav\n" +
				"execute-app-name: playback",
		},
		{
			cmd("sendmsg").WithMessage(map[string]string{
				"call-command":     "set",
				"event-lock":       "true",
				"execute-app-name": "foo=bar",
			}, ""),
			"sendmsg\n" +
				"call-command: set\n" +
				"event-lock: true\n" +
				"execute-app-name: foo=bar",
		},
		{
			cmd("sendmsg").WithMessage(map[string]string{
				"call-command":     "execute",
				"execute-app-name": "set",
				"execute-app-arg":  "foo=bar",
				"event-lock":       "true",
			}, ""),
			"sendmsg\n" +
				"call-command: execute\n" +
				"event-lock: true\n" +
				"execute-app-arg: foo=bar\n" +
				"execute-app-name: set",
		},
		// spell-checker:enable
	}

	for i, tc := range tests {
		got := tc.command
		if got.String() != tc.want {
			t.Errorf("[%d] got: %s,\nwant: %s", i, got, tc.want)
		}

		slog.Info("esl: test", slog.Any("cmd", got))
	}
}
