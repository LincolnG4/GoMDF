package mf4

import (
	"errors"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

var (
	// ErrNotMDF4 is returned by Open for files that are not MDF >= 4.00.
	ErrNotMDF4 = errors.New("not an MDF 4.x file")
	// ErrUnfinalized is returned for files whose writer did not finalize
	// them; reading would yield undefined data.
	ErrUnfinalized = errors.New("unfinalized MDF file")
	// ErrChannelNotFound is returned when a channel name cannot be
	// resolved.
	ErrChannelNotFound = errors.New("channel not found")
	// ErrUnsupported is returned for valid MDF features this module does
	// not implement yet (e.g. MDF 4.2 column-oriented storage).
	ErrUnsupported = errors.New("unsupported MDF feature")
)

// BlockError describes a structural error in a specific block; it wraps
// the block ID and absolute file offset.
type BlockError = blocks.BlockError
