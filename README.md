# ESL

Another library for working with the FreeSWITCH server on Golang.

In this version, only the client connection is supported, and the outbound connection has not yet 
been implemented, because it simply was not necessary.

```golang
// initialize buffered events channel
events := make(chan esl.Event, 1)
// read events
go func() {
    for ev := range events {
        fmt.Println(ev.Name(), ev.Get("Job-UUID"))
    }
}()

// connect to FreeSWITCH & init events channel with auto-close flag
client, err := esl.Connect("10.10.61.76", "ClueCon",
    esl.WithEvents(events, true))
if err != nil {
    panic(err)
}
defer client.Close()

// send a command
msg, err := client.API("show calls count")
if err != nil {
    panic(err)
}
fmt.Println(msg)

// subscribe to BACKGROUND_JOB events
if err = client.Subscribe("BACKGROUND_JOB"); err != nil {
    panic(err)
}

// send a background command
if err = client.JobWithID("uptime s", "test-xxx"); err != nil {
    panic(err)
}
```