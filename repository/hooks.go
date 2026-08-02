package repository

// OnLogSaved is set by the http package at startup so repository/mysql can
// notify connected clients (Socket.IO) without importing http, which would
// create an import cycle (http already imports repository/mysql). nil until
// the server wires it up (e.g. in tests / before http.Start()).
var OnLogSaved func(interface{})
