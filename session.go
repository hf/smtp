package smtp

import (
	"bytes"
	"context"
	"go.uber.org/zap"
)

var (
	extensionsTLS   = "250-8BITMIME\r\n250-SIZE\r\n250 STARTTLS\r\n"
	extensionsNoTLS = "250-8BITMIME\r\n250 SIZE\r\n"
)

type envelopeState = int

const (
	envelopeBlank      envelopeState = iota
	envelopeCreated                  = iota
	envelopeRecipients               = iota
	envelopeData                     = iota
)

type sessionState struct {
	domain []byte
	tls    bool

	env      Envelope
	envState envelopeState
}

func (st sessionState) inEHLO() bool {
	return nil != st.domain
}

func (st sessionState) inSTARTTLS() bool {
	return st.tls && st.inEHLO()
}

func (st sessionState) inMAIL() bool {
	return envelopeCreated == st.envState
}

func (st sessionState) inRCPT() bool {
	return envelopeRecipients == st.envState
}

func (st sessionState) inDATA() bool {
	return envelopeData == st.envState
}

func (st sessionState) Discard(ctx context.Context) error {
	env := st.env

	st.env = nil
	st.envState = envelopeBlank

	if nil != env {
		return env.Discard(ctx)
	}

	return nil
}

type sessionConfig struct {
	domain string

	tls         bool
	tlsRequired bool

	newEnvelope func(ctx context.Context, sess *Session) (Envelope, error)

	logger *zap.Logger
}

// An SMTP session with a client.
type Session struct {
	// Unique ID of this session.
	ID string

	// Address of the SMTP client.
	Addr string

	config sessionConfig

	state sessionState
}

// Domain reported by the SMTP client in the EHLO/HELO command. Will be nil if
// such a command has not been received.
func (sess *Session) Domain() []byte {
	return sess.state.domain
}

// Whether the session is over a TLS connection.
func (sess *Session) ViaTLS() bool {
	return sess.state.tls
}

type sessionAction = uint

const (
	keepSession    sessionAction = iota
	closeSession                 = iota
	upgradeSession               = iota
)

func (sess *Session) greet(ctx context.Context) []byte {
	return replyServiceReady(sess.config.domain)
}

func (sess *Session) kill(ctx context.Context) ([]byte, error) {
	return replyServiceNotAvailable(sess.config.domain), sess.state.Discard(ctx)
}

func (sess *Session) advance(ctx context.Context, line []byte) ([]byte, sessionAction, error) {
	if sess.state.inDATA() {
		return sess.processContent(ctx, line)
	} else {
		return sess.processCommand(ctx, line)
	}
}

var (
	endOfData       = []byte(".\r\n")
	escapeDotPrefix = []byte("..")
)

func (sess *Session) processContent(ctx context.Context, line []byte) ([]byte, sessionAction, error) {
	if bytes.Equal(line, endOfData) {
		action, err := sess.state.env.Commit(ctx)
		if nil != err {
			sess.config.logger.Warn("commit failed", zap.Error(err))

			return replyDATATransactionFailed, keepSession, err
		}

		sess.state.env = nil
		sess.state.envState = envelopeBlank

		switch action {
		case AcceptCommit:
			break

		case RejectCommitPermanently:
			return replyDATARejectPermanent, keepSession, err

		case RejectCommitForTooManyRecipients:
			return replyDATARejectNumberRecipients, keepSession, err

		case RejectCommitTemporarilyForSizeExceeded:
			return replyDATARejectSizeTemporary, keepSession, err

		case RejectCommitPermanentlyForSizeExceeded:
			return replyDATARejectSizePermanent, keepSession, err

		default:
			return replyDATARejectTemporary, keepSession, err
		}

		return replyAnyOk, keepSession, err
	} else {
		write := line

		if bytes.HasPrefix(line, escapeDotPrefix) {
			write = line[1:]
		}

		err := sess.state.env.Write(ctx, write)

		if nil != err {
			sess.config.logger.Warn("adding new line to envelope failed", zap.Error(err))

			return replyServiceNotAvailable(sess.config.domain), closeSession, err
		}

		return nil, keepSession, nil
	}
}

func (sess *Session) processCommand(ctx context.Context, line []byte) ([]byte, sessionAction, error) {
	command, result := parseCommand(line)

	switch result {
	case parseBadFormat:
		return replyAnyBadCommand, keepSession, nil
	case parseUnrecognizedCommand:
		return replyAnyBadCommand, keepSession, nil
	}

	if sess.config.tlsRequired && sess.config.tls && !sess.state.inSTARTTLS() {
		switch command.name {
		case commandHELO, commandEHLO:
			return sess.processEHLO(ctx, command)

		case commandSTARTTLS:
			return sess.processSTARTTLS(ctx, command)

		default:
			return replySTARTTLSRequired, keepSession, nil
		}
	}

	switch command.name {
	case commandMAIL:
		return sess.processMAIL(ctx, command)
	case commandRSET:
		return sess.processRSET(ctx, command)
	case commandQUIT:
		return sess.processQUIT(ctx, command)
	case commandNOOP:
		return sess.processNOOP(ctx, command)
	case commandSTARTTLS:
		return sess.processSTARTTLS(ctx, command)

	case commandHELO, commandEHLO:
		return sess.processEHLO(ctx, command)

	case commandHELP:
		return sess.processHELP(ctx, command)
	case commandEXPN:
		return sess.processEXPN(ctx, command)
	case commandVRFY:
		return sess.processVRFY(ctx, command)
	}

	if sess.state.inMAIL() {
		switch command.name {
		case commandRCPT:
			return sess.processRCPT(ctx, command)
		}
	} else if sess.state.inRCPT() {
		switch command.name {
		case commandRCPT:
			return sess.processRCPT(ctx, command)
		case commandDATA:
			return sess.processDATA(ctx, command)
		}
	}

	return replyAnyBadSequence, keepSession, nil
}

func (sess *Session) processNOOP(ctx context.Context, command command) ([]byte, sessionAction, error) {
	return replyAnyOk, keepSession, nil
}

func (sess *Session) processQUIT(ctx context.Context, command command) ([]byte, sessionAction, error) {
	return replyServiceClosing(sess.config.domain), closeSession, sess.state.Discard(ctx)
}

func (sess *Session) processRSET(ctx context.Context, command command) ([]byte, sessionAction, error) {
	err := sess.state.Discard(ctx)
	if nil != err {
		sess.config.logger.Warn("discarding state for transaction reset failed", zap.Error(err))
	}

	return replyAnyOk, keepSession, err
}

func (sess *Session) processMAIL(ctx context.Context, command command) ([]byte, sessionAction, error) {
	if nil == command.addr {
		return replyAnyBadCommand, keepSession, nil
	}

	err := sess.state.Discard(ctx)
	if nil != err {
		sess.config.logger.Warn("discarding state for new transaction failed", zap.Error(err))
	}

	env, err := sess.config.newEnvelope(ctx, sess)
	if nil != err {
		return replyServiceNotAvailable(sess.config.domain), closeSession, err
	}

	fromAction, err := env.From(ctx, command.addr)
	if nil != err {
		sess.config.logger.Warn("adding reverse-path failed", zap.Error(err))

		return replyServiceNotAvailable(sess.config.domain), closeSession, env.Discard(ctx)
	}

	switch fromAction {
	case AcceptFROM:
		break

	case RejectFROMPermanently:
		return replyMAILRejectFROMPermanent, keepSession, env.Discard(ctx)

	default:
		return replyMAILRejectFROMTemporary, keepSession, env.Discard(ctx)
	}

	sizeAction, err := env.Size(ctx, command.sizeHint)
	if nil != err {
		sess.config.logger.Warn("adding size-hint failed", zap.Error(err))

		return replyServiceNotAvailable(sess.config.domain), closeSession, env.Discard(ctx)
	}

	switch sizeAction {
	case AcceptSIZE:
		break

	case RejectSIZEPermanently:
		return replyMAILRejectSIZEPermanent, keepSession, env.Discard(ctx)

	default:
		return replyMAILRejectSIZETemporary, keepSession, env.Discard(ctx)
	}

	sess.state.env = env
	sess.state.envState = envelopeCreated

	return replyAnyOk, keepSession, nil
}

func (sess *Session) processRCPT(ctx context.Context, command command) ([]byte, sessionAction, error) {
	if nil == command.addr {
		return replyAnyBadCommand, keepSession, nil
	}

	action, err := sess.state.env.To(ctx, command.addr)
	if nil != err {
		sess.config.logger.Warn("adding recipient failed", zap.Error(err))

		return replyServiceNotAvailable(sess.config.domain), closeSession, sess.state.Discard(ctx)
	}

	switch action {
	case AcceptTO:
		break

	case RejectTOPermanently:
		return replyRCPTRejectPermanent, keepSession, err

	default:
		return replyRCPTRejectTemporary, keepSession, err
	}

	sess.state.envState = envelopeRecipients

	return replyAnyOk, keepSession, err
}

func (sess *Session) processDATA(ctx context.Context, command command) ([]byte, sessionAction, error) {
	action, err := sess.state.env.Open(ctx)
	if nil != err {
		sess.config.logger.Warn("open failed", zap.Error(err))
		return replyDATATransactionFailed, keepSession, sess.state.Discard(ctx)
	}

	switch action {
	case AcceptDATA:
		break

	default:
		return replyDATATransactionFailed, keepSession, sess.state.Discard(ctx)
	}

	sess.state.envState = envelopeData

	return replyDATAContinue, keepSession, err
}

func (sess *Session) processSTARTTLS(ctx context.Context, command command) ([]byte, sessionAction, error) {
	if sess.state.tls || !sess.config.tls {
		return replyAnyNotImplemented, keepSession, nil
	}

	sess.state.domain = nil
	sess.state.tls = true

	return replySTARTTLSReady, upgradeSession, sess.state.Discard(ctx)
}

func (sess *Session) processEHLO(ctx context.Context, command command) ([]byte, sessionAction, error) {
	if nil == command.addr {
		return replyAnyBadCommand, keepSession, nil
	}

	err := sess.state.Discard(ctx)

	sess.state.domain = command.addr

	if commandHELO == command.name {
		return replyEHLOOk(sess.config.domain, ""), keepSession, err
	}

	extensions := extensionsTLS

	if sess.state.tls || !sess.config.tls {
		extensions = extensionsNoTLS
	}

	return replyEHLOOk(sess.config.domain, extensions), keepSession, err
}

func (sess *Session) processHELP(ctx context.Context, command command) ([]byte, sessionAction, error) {
	return replyAnyNotImplemented, keepSession, nil
}

func (sess *Session) processEXPN(ctx context.Context, command command) ([]byte, sessionAction, error) {
	return replyAnyNotImplemented, keepSession, nil
}

func (sess *Session) processVRFY(ctx context.Context, command command) ([]byte, sessionAction, error) {
	return replyAnyNotImplemented, keepSession, nil
}
