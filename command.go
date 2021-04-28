package smtp

import (
	"bytes"
	"regexp"
	"strconv"
)

var patternCommand = regexp.MustCompile("(?i)^([A-Z]+)(\r\n$| +\r\n$| +(.*) *\r\n$)")

type parseResult = int

const (
	parseOk                  parseResult = iota
	parseBadFormat                       = iota
	parseUnrecognizedCommand             = iota
)

type commandName = int

const (
	commandHELO     commandName = iota
	commandEHLO                 = iota
	commandRCPT                 = iota
	commandMAIL                 = iota
	commandDATA                 = iota
	commandRSET                 = iota
	commandQUIT                 = iota
	commandEXPN                 = iota
	commandVRFY                 = iota
	commandHELP                 = iota
	commandNOOP                 = iota
	commandSTARTTLS             = iota
)

type command struct {
	name commandName

	addr     []byte
	sizeHint uint64
}

var commandParsers = map[string]func(args []byte) command{
	"HELO":     parseHELO,
	"EHLO":     parseEHLO,
	"RCPT":     parseRCPT,
	"MAIL":     parseMAIL,
	"DATA":     parseDATA,
	"RSET":     parseRSET,
	"QUIT":     parseQUIT,
	"EXPN":     parseEXPN,
	"VRFY":     parseVRFY,
	"HELP":     parseHELP,
	"NOOP":     parseNOOP,
	"STARTTLS": parseSTARTTLS,
}

func parseCommand(line []byte) (command, parseResult) {
	parts := patternCommand.FindSubmatch(line)

	if nil == parts {
		return command{}, parseBadFormat
	}

	name := string(bytes.ToUpper(parts[1]))

	parser := commandParsers[name]
	if nil == parser {
		return command{}, parseUnrecognizedCommand
	}

	args := bytes.TrimSpace(parts[3])

	return parser(args), parseOk
}

var patternEHLO = regexp.MustCompile("(?i)([^ ]+)")

func parseHELO(args []byte) command {
	cmd := parseEHLO(args)
	cmd.name = commandHELO

	return cmd
}

func parseEHLO(args []byte) command {
	matches := patternEHLO.FindSubmatch(args)

	if nil == matches {
		return command{
			name: commandEHLO,
		}
	}

	addr := make([]byte, len(matches[1]))
	copy(addr, matches[1])

	return command{
		name: commandEHLO,
		addr: addr,
	}
}

var patternFROM = regexp.MustCompile("(?i)FROM:<([^>]+)>")
var patternSIZE = regexp.MustCompile("(?i)SIZE=([1-9][0-9]*|0)")

func parseMAIL(args []byte) command {
	cmd := command{
		name: commandMAIL,
	}

	matches := patternFROM.FindSubmatch(args)

	if nil != matches {
		cmd.addr = make([]byte, len(matches[1]))
		copy(cmd.addr, matches[1])
	}

	matches = patternSIZE.FindSubmatch(args)
	if nil != matches {
		sizeHint, err := strconv.ParseUint(string(matches[1]), 10, 64)
		if nil == err {
			cmd.sizeHint = sizeHint
		}
	}

	return cmd
}

var patternTO = regexp.MustCompile("(?i)TO:<([^>]+)>")

func parseRCPT(args []byte) command {
	cmd := command{
		name: commandRCPT,
	}

	matches := patternTO.FindSubmatch(args)

	if nil != matches {
		cmd.addr = make([]byte, len(matches[1]))
		copy(cmd.addr, matches[1])
	}

	return cmd
}

func parseDATA(args []byte) command {
	return command{
		name: commandDATA,
	}
}

func parseRSET(arsg []byte) command {
	return command{
		name: commandRSET,
	}
}

func parseQUIT(arsg []byte) command {
	return command{
		name: commandQUIT,
	}
}

func parseNOOP(args []byte) command {
	return command{
		name: commandNOOP,
	}
}

func parseHELP(args []byte) command {
	return command{
		name: commandHELP,
	}
}

func parseEXPN(args []byte) command {
	return command{
		name: commandEXPN,
	}
}

func parseVRFY(args []byte) command {
	return command{
		name: commandVRFY,
	}
}

func parseSTARTTLS(args []byte) command {
	return command{
		name: commandSTARTTLS,
	}
}
