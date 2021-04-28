package smtp

var (
	replyAnyOk               = []byte("250 Requested mail action okay, completed\r\n")
	replyAnyBadCommand       = []byte("500 Syntax error, command unrecognized\r\n")
	replyAnyBadSequence      = []byte("503 Bad sequence of commands\r\n")
	replyAnyNotImplemented   = []byte("502 Command not implemented\r\n")
	replyAnyTemporaryFailure = []byte("421 Temporary failure\r\n")
)

var (
	replyMAILRejectFROMPermanent = []byte("550 Requested action not taken: sender is blocked\r\n")
	replyMAILRejectFROMTemporary = []byte("450 Requested mail action not taken: temporarily blocked\r\n")
	replyMAILRejectSIZEPermanent = []byte("552 message size exceeds fixed maximium message size\r\n")
	replyMAILRejectSIZETemporary = []byte("452 insufficient system storage\r\n")
)

var (
	replyRCPTRejectPermanent = []byte("550 Requested action not taken: mailbox unavailable\r\n")
	replyRCPTRejectTemporary = []byte("450 Requested mail action not taken: mailbox unavailable\r\n")
)

var (
	replyDATAContinue               = []byte("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
	replyDATATransactionFailed      = []byte("554 Transaction failed\r\n")
	replyDATARejectNumberRecipients = []byte("452 Requested action not taken: too many recipients\r\n")
	replyDATARejectPermanent        = []byte("550 Requested action not taken: mailbox unavailable\r\n")
	replyDATARejectTemporary        = []byte("450 Requested mail action not taken: mailbox unavailable\r\n")
	replyDATARejectSizePermanent    = []byte("552 Requested mail action aborted: exceeded storage allocation\r\n")
	replyDATARejectSizeTemporary    = []byte("452 Requested action not taken: insufficient system storage\r\n")
)

var (
	replySTARTTLSReady       = []byte("220 Ready to start TLS\r\n")
	replySTARTTLSUnavailable = []byte("454 TLS not available due to temporary reason\r\n")
	replySTARTTLSRequired    = []byte("530 Must issue a STARTTLS command first\r\n")
)

func replyServiceReady(domain string) []byte {
	return []byte("220 " + domain + " Service ready\r\n")
}

func replyServiceClosing(domain string) []byte {
	return []byte("221 " + domain + " Service closing transmission channel\r\n")
}

func replyServiceNotAvailable(domain string) []byte {
	return []byte("421 " + domain + " Service not available, closing transmission channel\r\n")
}

func replyEHLOOk(domain string, extensions string) []byte {
	if "" == extensions {
		return []byte("250 " + domain + " greetings\r\n")
	}

	return []byte("250-" + domain + " greetings\r\n" + extensions)
}
