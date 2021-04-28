package smtp

import (
	"context"
)

type FromAction = int

const (
	AcceptFROM            FromAction = iota
	RejectFROMTemporarily            = iota
	RejectFROMPermanently            = iota
)

type ToAction = int

const (
	AcceptTO            ToAction = iota
	RejectTOTemporarily          = iota
	RejectTOPermanently          = iota
)

type SizeAction = int

const (
	AcceptSIZE            SizeAction = iota
	RejectSIZETemporarily            = iota
	RejectSIZEPermanently            = iota
)

type DataAction = int

const (
	AcceptDATA DataAction = iota
	RejectDATA            = iota
)

type CommitAction = int

const (
	AcceptCommit                           CommitAction = iota
	RejectCommitTemporarily                             = iota
	RejectCommitPermanently                             = iota
	RejectCommitForTooManyRecipients                    = iota
	RejectCommitTemporarilyForSizeExceeded              = iota
	RejectCommitPermanentlyForSizeExceeded              = iota
)

// Describes a SMTP mail envelope.
type Envelope interface {
	// Add the reverse path to the envelope. Returning an error will terminate
	// the connection.
	From(ctx context.Context, addr []byte) (FromAction, error)

	// Add a size hint to the envelope. Returning an error will terminate the
	// connection. Size hint of 0 may mean an advertised data length of 0 or
	// none advertised.
	Size(ctx context.Context, size uint64) (SizeAction, error)

	// Add a recipient to the envelope. Returning an error will terminate the
	// connection.
	To(ctx context.Context, addr []byte) (ToAction, error)

	// Open the envelope for writing data. Ideally report any errors from From,
	// Size or To in this step, or from the context cancellation, as an error
	// here does not terminate the connection.
	Open(ctx context.Context) (DataAction, error)

	// Write a line to the envelope. Returning an error will terminate the
	// connection.
	Write(ctx context.Context, line []byte) error

	// Commit the data. If you accept the commit, the SMTP client expects the
	// mail to be delivered. If you return an error the transaction will be
	// cancelled.
	Commit(ctx context.Context) (CommitAction, error)

	// Discard this envelope. Returning an error does not affect the
	// connection.
	Discard(ctx context.Context) error
}
