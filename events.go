package esl

import (
	"bytes"
	"strings"
)

// eventNames is a map that contains the predefined names of various events as keys.
//
//nolint:gochecknoglobals
var eventNames = map[string]struct{}{
	// spell-checker:disable
	"CUSTOM":                   {},
	"CLONE":                    {},
	"CHANNEL_CREATE":           {},
	"CHANNEL_DESTROY":          {},
	"CHANNEL_STATE":            {},
	"CHANNEL_CALLSTATE":        {},
	"CHANNEL_ANSWER":           {},
	"CHANNEL_HANGUP":           {},
	"CHANNEL_HANGUP_COMPLETE":  {},
	"CHANNEL_EXECUTE":          {},
	"CHANNEL_EXECUTE_COMPLETE": {},
	"CHANNEL_HOLD":             {},
	"CHANNEL_UNHOLD":           {},
	"CHANNEL_BRIDGE":           {},
	"CHANNEL_UNBRIDGE":         {},
	"CHANNEL_PROGRESS":         {},
	"CHANNEL_PROGRESS_MEDIA":   {},
	"CHANNEL_OUTGOING":         {},
	"CHANNEL_PARK":             {},
	"CHANNEL_UNPARK":           {},
	"CHANNEL_APPLICATION":      {},
	"CHANNEL_ORIGINATE":        {},
	"CHANNEL_UUID":             {},
	"API":                      {},
	"LOG":                      {},
	"INBOUND_CHAN":             {},
	"OUTBOUND_CHAN":            {},
	"STARTUP":                  {},
	"SHUTDOWN":                 {},
	"PUBLISH":                  {},
	"UNPUBLISH":                {},
	"TALK":                     {},
	"NOTALK":                   {},
	"SESSION_CRASH":            {},
	"MODULE_LOAD":              {},
	"MODULE_UNLOAD":            {},
	"DTMF":                     {},
	"MESSAGE":                  {},
	"PRESENCE_IN":              {},
	"NOTIFY_IN":                {},
	"PRESENCE_OUT":             {},
	"PRESENCE_PROBE":           {},
	"MESSAGE_WAITING":          {},
	"MESSAGE_QUERY":            {},
	"ROSTER":                   {},
	"CODEC":                    {},
	"BACKGROUND_JOB":           {},
	"DETECTED_SPEECH":          {},
	"DETECTED_TONE":            {},
	"PRIVATE_COMMAND":          {},
	"HEARTBEAT":                {},
	"TRAP":                     {},
	"ADD_SCHEDULE":             {},
	"DEL_SCHEDULE":             {},
	"EXE_SCHEDULE":             {},
	"RE_SCHEDULE":              {},
	"RELOADXML":                {},
	"NOTIFY":                   {},
	"PHONE_FEATURE":            {},
	"PHONE_FEATURE_SUBSCRIBE":  {},
	"SEND_MESSAGE":             {},
	"RECV_MESSAGE":             {},
	"REQUEST_PARAMS":           {},
	"CHANNEL_DATA":             {},
	"GENERAL":                  {},
	"COMMAND":                  {},
	"SESSION_HEARTBEAT":        {},
	"CLIENT_DISCONNECTED":      {},
	"SERVER_DISCONNECTED":      {},
	"SEND_INFO":                {},
	"RECV_INFO":                {},
	"RECV_RTCP_MESSAGE":        {},
	"SEND_RTCP_MESSAGE":        {},
	"CALL_SECURE":              {},
	"NAT":                      {},
	"RECORD_START":             {},
	"RECORD_STOP":              {},
	"PLAYBACK_START":           {},
	"PLAYBACK_STOP":            {},
	"CALL_UPDATE":              {},
	"FAILURE":                  {},
	"SOCKET_DATA":              {},
	"MEDIA_BUG_START":          {},
	"MEDIA_BUG_STOP":           {},
	"CONFERENCE_DATA_QUERY":    {},
	"CONFERENCE_DATA":          {},
	"CALL_SETUP_REQ":           {},
	"CALL_SETUP_RESULT":        {},
	"CALL_DETAIL":              {},
	"DEVICE_STATE":             {},
	"TEXT":                     {},
	"SHUTDOWN_REQUESTED":       {},
	// spell-checker:enable
}

const eventAll = "all"

// buildEventNamesCmd builds a command string for enabling/disabling FreeSWITCH
// event types. It accepts a list of event names and returns a command string
// suitable for passing to the 'event_subscribe' API.
func buildEventNamesCmd(names ...string) string {
	if len(names) == 0 || names[0] == "" || strings.EqualFold(names[0], eventAll) {
		return eventAll
	}

	var (
		native   strings.Builder
		custom   bytes.Buffer
		isCustom bool
	)

	for _, name := range names {
		switch {
		case name == "":
			continue
		case strings.EqualFold(name, eventAll):
			return eventAll
		case strings.EqualFold(name, "CUSTOM"):
			isCustom = true

			continue
		default:
			if _, ok := eventNames[name]; ok {
				if native.Len() > 0 {
					native.WriteByte(' ')
				}

				native.WriteString(name)

				continue
			}

			// custom event name
			if custom.Len() > 0 {
				custom.WriteByte(' ')
			}

			custom.WriteString(strings.TrimPrefix(name, "CUSTOM "))
		}
	}

	// join event names
	if isCustom || custom.Len() > 0 {
		if native.Len() > 0 {
			native.WriteByte(' ')
		}

		native.WriteString("CUSTOM")

		if custom.Len() > 0 {
			native.WriteByte(' ')
			custom.WriteTo(&native) //nolint:errcheck // ignore write error
		}
	}

	if native.Len() == 0 {
		return eventAll
	}

	return native.String()
}
