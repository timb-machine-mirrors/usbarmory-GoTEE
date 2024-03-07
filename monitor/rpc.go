// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package monitor

import (
	"errors"
	"fmt"
	"net/rpc/jsonrpc"
	errno "syscall"

	"github.com/usbarmory/GoTEE/syscall"
)

// Read reads up to len(p) bytes into p. The read data is received from the
// execution context memory, after it is being written with syscall.Write().
func (ctx *ExecCtx) Read(p []byte) (int, error) {
	r := len(ctx.in)
	n := len(p)

	switch {
	case r <= 0:
		return 0, nil
	case r < n:
		n = r
	}

	copy(p, ctx.in[0:n])
	ctx.in = ctx.in[n:]

	return n, nil
}

// Write writes len(p) bytes from p to the underlying data stream, it never
// returns an error. The written data is buffered within the execution context,
// waiting for its read through syscall.Read().
func (ctx *ExecCtx) Write(p []byte) (int, error) {
	ctx.out = append(ctx.out, p...)
	return len(p), nil
}

// Close has no effect.
func (ctx *ExecCtx) Close() error {
	return nil
}

// Recv handles syscall.Write() as received from the execution context memory,
// the written data is buffered (see Read()).
func (ctx *ExecCtx) Recv() error {
	off := ctx.A1() - ctx.Memory.Start()
	n := ctx.A2()

	if !(off >= 0 && off < (ctx.Memory.Size()-n)) {
		return errors.New("invalid offset")
	}

	buf := make([]byte, n)

	ctx.Memory.Read(ctx.Memory.Start(), int(off), buf)
	ctx.in = append(ctx.in, buf...)

	return nil
}

// Flush handles syscall.Read() as received from the execution context, the
// buffered data (see Write()) is returned to the execution context memory..
func (ctx *ExecCtx) Flush(err error) (int, error) {
	off := ctx.A1() - ctx.Memory.Start()
	n := ctx.A2()
	r := uint(len(ctx.out))

	if !(off >= 0 && off < (ctx.Memory.Size()-n)) {
		return 0, errors.New("invalid offset")
	}

	switch {
	case err != nil:
		ctx.Ret(-(int(errno.EPIPE)))
		return 0, nil
	case r <= 0:
		ctx.Ret(0)
		return 0, nil
	case r < n:
		n = r
	}

	ctx.Memory.Write(ctx.Memory.Start(), int(off), ctx.out[0:n])
	ctx.Ret(n)

	ctx.out = ctx.out[n:]

	return int(n), nil
}

func (ctx *ExecCtx) rpc() (err error) {
	switch num := ctx.A0(); num {
	case syscall.SYS_RPC_REQ:
		if err = ctx.Recv(); err != nil {
			return
		}

		err = ctx.Server.ServeRequest(jsonrpc.NewServerCodec(ctx))
	case syscall.SYS_RPC_RES:
		_, err = ctx.Flush(nil)
	default:
		err = fmt.Errorf("invalid syscall %d", num)
	}

	return
}
