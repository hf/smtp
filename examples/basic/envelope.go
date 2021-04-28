package main

import (
	"bytes"
	"context"
	"github.com/hf/smtp"
	"go.uber.org/zap"
)

type exampleEnvelope struct {
	logger *zap.Logger

	from       []byte
	recipients [][]byte
	sizeHint   uint64

	data *bytes.Buffer
}

func (env *exampleEnvelope) From(ctx context.Context, addr []byte) (smtp.FromAction, error) {
	env.from = addr

	return smtp.AcceptFROM, nil
}

func (env *exampleEnvelope) To(ctx context.Context, addr []byte) (smtp.ToAction, error) {
	if nil == env.recipients {
		env.recipients = make([][]byte, 0, 10)
	}

	env.recipients = append(env.recipients, addr)

	return smtp.AcceptTO, nil
}

func (env *exampleEnvelope) Size(ctx context.Context, sizeHint uint64) (smtp.SizeAction, error) {
	env.sizeHint = sizeHint

	return smtp.AcceptSIZE, nil
}

func (env *exampleEnvelope) Open(ctx context.Context) (smtp.DataAction, error) {
	env.data = bytes.NewBuffer(make([]byte, 0, 8*1024))

	return smtp.AcceptDATA, nil
}

func (env *exampleEnvelope) Write(ctx context.Context, line []byte) error {
	env.data.Write(line)

	return nil
}

func (env *exampleEnvelope) Commit(ctx context.Context) (smtp.CommitAction, error) {
	env.logger.Debug("received mail",
		zap.ByteString("data", env.data.Bytes()),
		zap.ByteString("from", env.from),
		zap.ByteString("to", bytes.Join(env.recipients, []byte(", "))))

	return smtp.AcceptCommit, nil
}

func (env *exampleEnvelope) Discard(ctx context.Context) error {
	return nil
}
