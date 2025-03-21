package gconfig

import (
	"context"
	"net/url"
)

func NewMockByURL(addr string) (Configure, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	data := make(map[string]string)
	for k, v := range q {
		data[k] = v[0]
	}

	return NewMock(data), nil
}

func NewMock(data map[string]string) Configure {
	return &mockConfigure{data: data, listeners: map[string]Listener{}}
}

// mockConfigure 仅用于单测,不区分group
type mockConfigure struct {
	data      map[string]string
	listeners map[string]Listener
}

func (c *mockConfigure) Get(ctx context.Context, key string, opts ...Option) (string, error) {
	data, ok := c.data[key]
	if !ok {
		return "", ErrNotFound
	}

	return data, nil
}

func (c *mockConfigure) Put(ctx context.Context, key string, value string, opts ...Option) error {
	if c.data == nil {
		c.data = make(map[string]string)
	}
	c.data[key] = value
	c.onChange(key, value)
	return nil
}

func (c *mockConfigure) Delete(ctx context.Context, key string, opts ...Option) error {
	if _, ok := c.data[key]; ok {
		delete(c.data, key)
		c.onChange(key, "")
	}
	return nil
}

func (c *mockConfigure) Listen(ctx context.Context, key string, listener Listener, opts ...Option) error {
	return c.doListen(key, listener)
}

func (c *mockConfigure) doListen(key string, listener Listener) error {
	if _, ok := c.listeners[key]; ok {
		return ErrDuplicateListen
	}

	c.listeners[key] = listener
	return nil
}

func (c *mockConfigure) onChange(key string, value string) {
	if l, ok := c.listeners[key]; ok {
		l.OnEvent(&Event{Type: EventTypeUpdate, Key: key, Value: value})
	}
}
