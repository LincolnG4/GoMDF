// Package mf4 reads ASAM MDF 4.x (Measurement Data Format) files.
//
// Files are memory-mapped by default and decoded lazily: Open parses only
// the metadata tree (groups, channels, conversions); sample data is
// decoded on demand, per channel, into typed columnar slices.
//
//	f, err := mf4.Open("measurement.mf4")
//	if err != nil { ... }
//	defer f.Close()
//
//	ch, err := f.Channel("EngineSpeed")
//	sig, err := ch.Read()          // *Signal: typed samples + master time
//
// Large files can be read incrementally with ChannelGroup.Chunks or a
// sample window with WithRange.
package mf4
