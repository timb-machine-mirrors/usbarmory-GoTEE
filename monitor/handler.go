// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package monitor

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/usbarmory/GoTEE/syscall"
)

// SecureHandler is the default handler for exceptions raised by a secure
// execution context to handle supported GoTEE system calls.
func SecureHandler(ctx *ExecCtx) (err error) {
	switch num := ctx.A0(); num {
	case syscall.SYS_EXIT:
		return errors.New("exit")
	case syscall.SYS_WRITE:
		print(string(ctx.A1()))
	case syscall.SYS_NANOTIME:
		ctx.Ret(time.Now().UnixNano())
	case syscall.SYS_GETRANDOM:
		off := ctx.A1() - ctx.Memory.Start()
		buf := make([]byte, ctx.A2())

		if !(off >= 0 && off < (ctx.Memory.Size()-uint(len(buf)))) {
			return errors.New("invalid read offset")
		}

		if n, err := rand.Read(buf); err != nil || n != int(ctx.A2()) {
			return errors.New("internal error")
		}

		ctx.Memory.Write(ctx.Memory.Start(), int(off), buf)
	case syscall.SYS_RPC_REQ, syscall.SYS_RPC_RES:
		if ctx.Server != nil {
			err = ctx.rpc()
		}
	default:
		err = fmt.Errorf("invalid syscall %d", num)
	}

	return
}

// NonSecureHandler is the default handler for exceptions raised by a
// non-secure execution context to handle supported GoTEE secure monitor calls.
func NonSecureHandler(ctx *ExecCtx) (err error) {
	// to be overridden by application
	log.Printf("NonSeureHandler: unimplemented")
	return
}