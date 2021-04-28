package smtp

import (
	"bytes"
	"context"
)

type testEnvelope struct {
	from       []byte
	recipients [][]byte
	sizeHint   uint64

	data *bytes.Buffer

	fromCalls    int
	sizeCalls    int
	toCalls      int
	openCalls    int
	writeCalls   int
	commitCalls  int
	discardCalls int

	onFrom func(ctx context.Context, env *testEnvelope, addr []byte) (FromAction, error)
	onSize func(ctx context.Context, env *testEnvelope, size uint64) (SizeAction, error)
	onTo   func(ctx context.Context, env *testEnvelope, addr []byte) (ToAction, error)

	onOpen func(ctx context.Context, env *testEnvelope) (DataAction, error)

	onWrite func(ctx context.Context, env *testEnvelope, line []byte) error

	onCommit  func(ctx context.Context, env *testEnvelope) (CommitAction, error)
	onDiscard func(ctx context.Context, env *testEnvelope) error
}

func (env *testEnvelope) From(ctx context.Context, addr []byte) (FromAction, error) {
	env.fromCalls += 1

	if nil != env.onFrom {
		return env.onFrom(ctx, env, addr)
	}

	env.from = addr

	return AcceptFROM, nil
}

func (env *testEnvelope) To(ctx context.Context, addr []byte) (ToAction, error) {
	env.toCalls += 1

	if nil != env.onTo {
		return env.onTo(ctx, env, addr)
	}

	if nil == env.recipients {
		env.recipients = make([][]byte, 0, 10)
	}

	env.recipients = append(env.recipients, addr)

	return AcceptTO, nil
}

func (env *testEnvelope) Size(ctx context.Context, sizeHint uint64) (SizeAction, error) {
	env.sizeCalls += 1

	if nil != env.onSize {
		return env.onSize(ctx, env, sizeHint)
	}

	env.sizeHint = sizeHint

	return AcceptSIZE, nil
}

func (env *testEnvelope) Open(ctx context.Context) (DataAction, error) {
	env.openCalls += 1

	if nil != env.onOpen {
		return env.onOpen(ctx, env)
	}

	env.data = bytes.NewBuffer(make([]byte, 0, 8*1024))

	return AcceptDATA, nil
}

func (env *testEnvelope) Write(ctx context.Context, line []byte) error {
	env.writeCalls += 1

	if nil != env.onWrite {
		return env.onWrite(ctx, env, line)
	}

	env.data.Write(line)

	return nil
}

func (env *testEnvelope) Commit(ctx context.Context) (CommitAction, error) {
	env.commitCalls += 1

	if nil != env.onCommit {
		return env.onCommit(ctx, env)
	}

	return AcceptCommit, nil
}

func (env *testEnvelope) Discard(ctx context.Context) error {
	env.discardCalls += 1

	if nil != env.onDiscard {
		return env.onDiscard(ctx, env)
	}

	env.from = nil
	env.recipients = nil
	env.data = nil

	return nil
}
