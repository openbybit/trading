package galert

import (
	"context"
	"fmt"
	"os"
)

type Level uint8

const (
	LevelInfo = Level(iota)
	LevelWarn
	LevelError
)

type Limter interface {
	Allow() bool
}

type Alerter interface {
	Alert(ctx context.Context, level Level, message string, opts ...Option)
}

type Config struct {
	Webhook    string   `json:"webhook" yaml:"webhook"`         //
	Title      string   `json:"title" yaml:"title"`             //
	Fields     []*Field `json:"fields" yaml:"fields"`           // 默认fields
	Footers    []*Field `json:"-" yaml:"-"`                     // default footers fields
	BufferSize int      `json:"buffer_size" yaml:"buffer_size"` // 异步发送buffer大小
}

func New(conf *Config) Alerter {
	if conf == nil {
		conf = &Config{
			Fields: DefaultFields(),
		}
	}

	if conf.BufferSize <= 0 {
		conf.BufferSize = 1024
	}

	if conf.Title == "" {
		conf.Title = "[Alert]"

		envName := os.Getenv("MY_ENV_NAME")
		if envName != "" {
			conf.Title += fmt.Sprintf("-[%s]", envName)
		}
	}

	a := &alert{
		title:    conf.Title,
		fields:   conf.Fields,
		footers:  conf.Footers,
		items:    make(chan *entry, conf.BufferSize),
		reporter: newLark(conf.Webhook),
	}

	go a.sendLoop()

	return a
}

type entry struct {
	level   Level
	title   string
	message string
	webhook string
	fields  []*Field
	footers []*Field
}

type reporter interface {
	Send(entry *entry) error
}

type alert struct {
	title    string
	fields   []*Field
	footers  []*Field
	items    chan *entry
	reporter reporter
}

func (a *alert) Alert(ctx context.Context, level Level, message string, opts ...Option) {
	o := Options{}
	for _, fn := range opts {
		fn(&o)
	}

	if o.limiter != nil && !o.limiter.Allow() {
		return
	}

	// check buffer is full
	if len(a.items) == cap(a.items) {
		return
	}

	item := &entry{}
	item.level = level
	item.message = message
	item.webhook = o.webhook
	item.title = o.title
	if item.title == "" {
		item.title = a.title
	}
	if len(a.fields) > 0 {
		item.fields = append(a.fields, o.fields...)
	} else {
		item.fields = o.fields
	}

	if len(a.footers) > 0 {
		item.footers = append(a.footers, o.footers...)
	} else {
		item.footers = o.footers
	}

	a.items <- item
}

func (a *alert) sendLoop() {
	for item := range a.items {
		_ = a.reporter.Send(item)
	}
}
