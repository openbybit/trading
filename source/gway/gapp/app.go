package gapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

const defaultAddress = ":6480"
const defaultExitTimeout = 30 * time.Second

type Logger interface {
	Printf(format string, args ...interface{})
}

type App interface {
	Run(opts ...Option) error
	Exit()
	getLogger() Logger
}

func New() App {
	return &app{}
}

type app struct {
	ctx    context.Context
	monit  monitor
	logger Logger
}

func (a *app) getLogger() Logger {
	return a.logger
}

func (a *app) Run(opts ...Option) error {
	o := Options{
		block:       blockOnSignal,
		addr:        defaultAddress,
		exitTimeout: defaultExitTimeout,
	}

	for _, fn := range opts {
		fn(&o)
	}

	if o.ctx == nil {
		o.ctx = context.Background()
	}

	if o.logger == nil {
		o.logger = log.New(os.Stdout, "", 0)
	}

	ctx, cancel := context.WithCancel(o.ctx)
	defer cancel()

	a.ctx = ctx
	a.logger = o.logger

	if err := trigger(a.ctx, LifecycleStart, o.hooks, true); err != nil {
		return err
	}

	a.monit.Start(o.addr, o.endpoints)
	o.block(a.ctx)

	cancelCh := make(chan error)
	go func() {
		errors := a.doExit(&o)
		if len(errors) > 0 {
			b := bytes.NewBuffer(nil)
			for _, e := range errors {
				b.WriteString(e.Error())
				b.WriteString("\n")
			}
			cancelCh <- fmt.Errorf("%s", b.String())
		} else {
			cancelCh <- nil
		}
	}()

	select {
	case err := <-cancelCh:
		return err
	case <-time.After(o.exitTimeout):
		return fmt.Errorf("app exit timeout, timeout=%v", o.exitTimeout)
	}
}

func (a *app) Exit() {
	a.ctx.Done()
}

func (a *app) doExit(o *Options) []error {
	var res []error
	if err := trigger(a.ctx, LifecycleStop, o.hooks, false); err != nil {
		log.Printf("lifecycle stop fail, err=%v\n", err)
		res = append(res, err)
	}

	if err := trigger(a.ctx, LifecycleShutdown, o.hooks, false); err != nil {
		log.Printf("lifecycle shutdown fail, err=%v\n", err)
		res = append(res, err)
	}

	if err := trigger(a.ctx, LifecycleDestroy, o.hooks, false); err != nil {
		log.Printf("lifecycle destory fail, err=%v\n", err)
		res = append(res, err)
	}

	a.monit.Stop()
	return res
}

func trigger(ctx context.Context, event LifecycleEvent, hooks []Lifecycle, interruptErr bool) error {
	var errs []error
	for _, h := range hooks {
		if err := h.OnLifecycle(ctx, event); err != nil {
			if interruptErr {
				return err
			}
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	for _, e := range errs {
		buf.WriteString(e.Error())
		buf.WriteByte('\n')
	}

	return errors.New(buf.String())
}
