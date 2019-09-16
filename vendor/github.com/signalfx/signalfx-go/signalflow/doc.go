// Package signalflow contains a SignalFx SignalFlow client, which can be used
// to execute analytics jobs against the SignalFx backend.
//
// The client currently only supports the execute request.  Not all SignalFlow
// messages are handled at this time, and some will be silently dropped.
//
// The client will automatically attempt to reconnect to the backend if the
// connection is broken.  There is a 5 second delay between retries.
package signalflow
