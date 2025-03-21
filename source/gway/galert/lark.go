package galert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// https://open.larksuite.com/open-apis/bot/v2/hook/5cf600a7-1fd8-4f6b-ad7b-57f0ab81ef5a
	defaultLarkDomain  = "https://open.larksuite.com"
	defaultLarkWebhook = "/open-apis/bot/v2/hook/5cf600a7-1fd8-4f6b-ad7b-57f0ab81ef5a"
)

const (
	larkTagHr        = "hr"
	larkTagDiv       = "div"
	larkTagMd        = "lark_md" // markdown
	larkTagPlainText = "plain_text"
)

const (
	maxLineSize = 50 // value长度超过此值会单独一行
)

type larkInform struct {
	MsgType string   `json:"msg_type"`
	Card    larkCard `json:"card"`
}

type larkCard struct {
	Config   larkConfig    `json:"config"`
	Header   larkHeader    `json:"header"`
	Elements []larkElement `json:"elements"`
}

type larkConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

type larkHeader struct {
	Template string   `json:"template"`
	Title    larkText `json:"title"`
}

type larkElement struct {
	Tag     string      `json:"tag,omitempty"`
	Content string      `json:"content,omitempty"`
	Fields  []larkField `json:"fields,omitempty"`
}

type larkField struct {
	IsShort bool      `json:"is_short,omitempty"`
	Text    *larkText `json:"text,omitempty"`
}

type larkText struct {
	Tag     string `json:"tag,omitempty"`
	Content string `json:"content,omitempty"`
}

func newLark(webhook string) reporter {
	if webhook == "" {
		webhook = defaultLarkWebhook
	}
	return &larkReporter{webhook: toLarkWebhook(webhook)}
}

// lark文档:
// https://open.larksuite.com/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/introduction
// https://open.larksuite.com/document/uAjLw4CM/ukTMukTMukTM/im-v1/message/create_json
// 页面搭建工具: https://open.larksuite.com/tool/cardbuilder?from=howtoguide
type larkReporter struct {
	webhook string
}

func (s *larkReporter) Send(item *entry) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	data, err := s.build(item)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}
	if item.webhook == "" {
		item.webhook = s.webhook
	}
	return sendPost(context.Background(), item.webhook, data)
}

func (s *larkReporter) build(item *entry) ([]byte, error) {
	const (
		interactiveMsgType = "interactive"
	)

	var colors = []string{
		LevelInfo:  "green",
		LevelWarn:  "yellow",
		LevelError: "red",
	}

	card := larkCard{
		Config: larkConfig{true, true},
		Header: larkHeader{
			Template: colors[item.level],
			Title: larkText{
				Content: item.title,
				Tag:     larkTagPlainText, // lark only support "plain_text"
			},
		},
	}

	fields := item.fields
	if item.message != "" {
		f := BasicField("message", item.message)
		fields = append(fields, f)
	}

	if len(fields) > 0 {
		card.Elements = append(card.Elements, s.buildFields(fields))
	}

	if len(item.footers) > 0 {
		card.Elements = append(card.Elements, larkElement{Tag: larkTagHr})
		card.Elements = append(card.Elements, s.buildFields(item.footers))
	}

	inform := larkInform{
		MsgType: interactiveMsgType,
		Card:    card,
	}

	return json.Marshal(inform)
}

func (s *larkReporter) buildFields(fields []*Field) larkElement {
	res := larkElement{Tag: larkTagDiv}
	for _, f := range fields {
		content := s.buildContent(f)
		if content == "" {
			continue
		}

		f1 := larkField{
			IsShort: f.isShort,
			Text: &larkText{
				Tag:     larkTagMd,
				Content: content,
			},
		}
		res.Fields = append(res.Fields, f1)
	}

	return res
}

func (s *larkReporter) buildContent(f *Field) string {
	var value string
	switch f.typ {
	case fieldTypeBasic:
		value = fmt.Sprintf("%v", f.value)
	case fieldTypeCurrentTime:
		layout, _ := f.value.(string)
		value = time.Now().Format(layout)
	case fieldTypeLink:
		info, _ := f.value.(*linkInfo)
		if info != nil {
			if info.Show != "" {
				value = fmt.Sprintf("[%s](%s)", info.Show, info.Link)
			} else {
				value = fmt.Sprintf("<%s>", info.Link)
			}
		}
	}

	if f.key == "" && f.value == "" {
		return ""
	}

	buf := strings.Builder{}
	if f.key != "" {
		buf.WriteString(fmt.Sprintf("**%s**:", f.key))
	}

	if len(value) > maxLineSize {
		buf.WriteString("\n")
		buf.WriteString(value)
	} else {
		buf.WriteString(" ")
		buf.WriteString(value)
	}

	return buf.String()
}

func toLarkWebhook(x string) string {
	if strings.HasPrefix(x, "https://") || strings.HasPrefix(x, "http://") {
		return x
	}

	if strings.HasPrefix(x, "/") {
		return defaultLarkDomain + x
	}

	return x
}
