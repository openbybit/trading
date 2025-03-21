package response

import (
	"bytes"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"bgw/pkg/common/util"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"
	"github.com/valyala/fasttemplate"
	"gopkg.in/yaml.v3"
)

func TestCommon(t *testing.T) {
	cl := CodeListener{}
	assert.Equal(t, 0, cl.GetPriority())
	assert.Nil(t, cl.GetEventType())
}

func TestOnEvent(t *testing.T) {
	cl := CodeListener{}
	err := cl.OnEvent(observer.NewBaseEvent(""))
	assert.NoError(t, err)

	err = cl.OnEvent(&observer.DefaultEvent{
		Key:   "bgw-errors-meta-xx",
		Value: "123",
	})
	assert.NoError(t, err)

	content := `
codes:
  - code: 1
    lang:
      en: "i am 1 msg"
      cn: "我是1的msg"
  - code: 2
    lang:
      en: "i am 2 msg{{.name}}"
      cn: "我是2的错误码{{.name}}"`
	err = cl.OnEvent(&observer.DefaultEvent{
		Key:   "bgw-errors-meta-xx",
		Value: content,
	})
	assert.NoError(t, err)

	err = cl.OnEvent(&observer.DefaultEvent{
		Key:   ErrorMatchPrefix,
		Value: "123",
	})
	assert.NoError(t, err)

	content = `
codes:
  - code: 1
    lang:
      en: "i am 1 msg"
      cn: "我是1的msg"
  - code: 2
    lang:
      en: "i am 2 msg{{.name}}"
      cn: "我是2的错误码{{.name}}"`
	err = cl.OnEvent(&observer.DefaultEvent{
		Key:   ErrorMatchPrefix,
		Value: content,
	})
	assert.NoError(t, err)

	err = cl.OnEvent(&observer.DefaultEvent{
		Key:   "123",
		Value: content,
	})
	assert.EqualError(t, err, "CodeListener unknown file: 123")
}

func TestCodeMatch(t *testing.T) {
	cl := CodeListener{}

	cl.matchs = make(map[string]*CodeMatch)
	cmm := &CodeMatch{
		Tag: 100,
	}
	cl.matchs[cl.formatMatch(1, 2, "123")] = cmm
	cl.specialMatchs = make(map[int64]*SpeCodeMatch)
	cl.specialMatchs[1] = &SpeCodeMatch{}
	cm := cl.codeMatch(1, 2, "123")
	assert.Equal(t, cmm, cm)

	cl.matchs[cl.formatMatch(1, 2, "")] = cmm
	delete(cl.matchs, cl.formatMatch(1, 2, "123"))
	cm = cl.codeMatch(1, 2, "123")
	assert.Equal(t, cmm, cm)

	delete(cl.matchs, cl.formatMatch(1, 2, "123"))
	delete(cl.matchs, cl.formatMatch(1, 2, ""))
	cm = cl.codeMatch(1, 2, "123")
	assert.Nil(t, cm)

	cl.matchs[cl.formatMatch(1, 0, "")] = cmm
	cm = cl.codeMatch(1, 0, "123")
	assert.Equal(t, cmm, cm)
}

func TestParseCodeMessage(t *testing.T) {
	cl := CodeListener{}

	c, m, b := cl.parseCodeMessage(context.Background(), nil, 0, "", 1, "1", 0, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, false, b)
	assert.Equal(t, "1", m)

	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "", 1, "1", MsgSource_OnlyMessage, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, false, b)
	assert.Equal(t, "1", m)

	cl.codes = make(map[string]*CodeMessage)
	cl.codes[cl.formatCode(1, 0)] = &CodeMessage{
		lang: map[string]*MessageTemplate{"en": {msg: "xxxx"}},
	}
	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "en", 1, "1", MsgSource_OnlyMessage, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, true, b)
	assert.Equal(t, "xxxx", m)

	cl.codes[cl.formatCode(1, 0)] = &CodeMessage{
		lang: map[string]*MessageTemplate{"en": {msg: "xxxx", template: fasttemplate.New("", "a", "b")}},
	}
	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "en", 1, "1", MsgSource_OnlyMessage, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, false, b)
	assert.Equal(t, "1", m)

	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "en", 1, "2", MsgSource_OnlyCode, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, false, b)
	assert.Equal(t, "2", m)

	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "en", 1, "2", MsgSource_Unknown, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, false, b)
	assert.Equal(t, "2", m)

	cl.codes[cl.formatCode(1, 0)] = &CodeMessage{
		lang: map[string]*MessageTemplate{"en": {msg: "xxxx"}},
	}
	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "en", 1, "2", MsgSource_FrontEnd, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, true, b)
	assert.Equal(t, "xxxx", m)
	cl.codes[cl.formatCode(1, 0)] = &CodeMessage{
		lang: map[string]*MessageTemplate{"en": {msg: "xxxx", template: fasttemplate.New("", "a", "b")}},
	}
	c, m, b = cl.parseCodeMessage(context.Background(), []byte("111"), 0, "en", 1, "2", MsgSource_FrontEnd, "")
	assert.Equal(t, int64(1), c)
	assert.Equal(t, false, b)
	assert.Equal(t, "2", m)
}

func Test_ParseCode(t *testing.T) {
	a := assert.New(t)
	content := `app: option
codes:
  - code: 1
    lang:
      en: "i am 1 msg"
      cn: "我是1的msg"
  - code: 2
    lang:
      en: "i am 2 msg{{.name}}"
      cn: "我是2的错误码{{.name}}"`

	eoption := &ErrorMetas{}
	err := yaml.Unmarshal([]byte(content), eoption)
	a.Nil(err)

	contentFuture := `app: future
codes:
  - code: 3
    lang:
      en: "i am 3 msg"
      cn: "我是1的msg"
  - code: 4
    lang:
      en: "i am 4 msg{{.name}}"
      cn: "我是4的错误码{{.name}}"`

	efuture := &ErrorMetas{}
	err = yaml.Unmarshal([]byte(contentFuture), efuture)
	a.Nil(err)

	setCodeLoader("future")
	loaderfuture, ok := codeLoader("future")
	a.True(ok)
	loaderfuture.buildMetas(efuture)

	setCodeLoader("option")
	loaderoption, ok := codeLoader("option")
	a.True(ok)
	loaderoption.buildMetas(eoption)

	m := loaderfuture.codeMessage("", 3, 0)
	a.NotNil(m)
	a.Equal("i am 3 msg", m.msg)

	moption := loaderoption.codeMessage("", 1, 0)
	a.NotNil(moption)
	a.Equal("i am 1 msg", moption.msg)
}

func Test_ParseCodeMatch(t *testing.T) {
	a := assert.New(t)
	content := `app: option
matchs:
  - target_code: 2
    source_code_list:
      - 1
`

	eoption := &ErrorMathcs{}
	err := yaml.Unmarshal([]byte(content), eoption)
	a.Nil(err)

	contentFuture := `app: future
matchs:
  - target_code: 4
    source_code_list:
      - 3
`

	efuture := &ErrorMathcs{}
	err = yaml.Unmarshal([]byte(contentFuture), efuture)
	a.Nil(err)

	setCodeLoader("future")
	loaderfuture, ok := codeLoader("future")
	a.True(ok)
	loaderfuture.buildMatchs(efuture)

	setCodeLoader("option")
	loaderoption, ok := codeLoader("option")
	a.True(ok)
	loaderoption.buildMatchs(eoption)

	matchfuture := loaderfuture.codeMatch(3, 0, "")
	a.NotNil(matchfuture)

	matchOption := loaderoption.codeMatch(1, 0, "")
	a.NotNil(matchOption)
}

func Test_codeListener_Variable(t *testing.T) {
	a := assert.New(t)

	setCodeLoader("future")
	loader, ok := codeLoader("future")
	a.True(ok)

	str := "i dont lejie"
	extra := []byte("{\"name\":\"tom\"}")
	tm := &MessageTemplate{
		template: fasttemplate.New(str, "{{.", "}}"),
		msg:      str,
	}
	msg, err := loader.Variable(tm, extra)
	a.NoError(err)
	a.Equal(str, msg)

	str = "i dont {{.action}}"
	extra = []byte(`{"action":"future"}`)
	tm = &MessageTemplate{
		template: fasttemplate.New(str, "{{.", "}}"),
		msg:      str,
	}
	msg, err = loader.Variable(tm, extra)
	a.NoError(err)
	a.Equal("i dont future", msg)

	str = "i {{.name}} dont {{.action}}"
	extra = []byte(`{"action":"future","name":"san"}`)
	tm = &MessageTemplate{
		template: fasttemplate.New(str, "{{.", "}}"),
		msg:      str,
	}
	msg, err = loader.Variable(tm, extra)
	a.NoError(err)
	a.Equal("i san dont future", msg)

	str = "i {{.name}} dont {{.action}}"
	extra = []byte(`{"action":"future","Name":"san"}`)
	tm = &MessageTemplate{
		template: fasttemplate.New(str, "{{.", "}}"),
		msg:      str,
	}
	msg, err = loader.Variable(tm, extra)
	a.NoError(err)
	a.Equal("i  dont future", msg)
}

func variable(message string, t *template.Template, extra []byte) (newMessage string, err error) {
	newMessage = message

	if idx := strings.Index(message, "{{."); idx < 0 {
		return
	}

	mm := make(map[string]string, 2)
	if len(extra) > 0 {
		if err = util.JsonUnmarshal(extra, &mm); err != nil {
			return
		}
	}

	buff := bytes.NewBuffer(make([]byte, 0, len(message)))
	err = t.Execute(buff, mm)
	if err != nil {
		return
	}
	newMessage = buff.String()
	return
}

func BenchmarkVariable(b *testing.B) {
	a := assert.New(b)

	str := "i {{.name}} dont {{.action}}"
	extra := []byte(`{"action":"future","name":"san"}`)
	t, err := MessageVariable.Parse(str)
	a.NoError(err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := variable(str, t, extra)
		a.NoError(err)
	}
}

func TestVariableWithByteMatch(t *testing.T) {
	a := assert.New(t)

	str := "i {{.name}} dont {{.action}}"
	extra := []byte(`{"action":"future","name":"san"}`)
	msg, err := variableWithByteMatch(str, extra)
	a.NoError(err)
	a.Equal("i san dont future", msg)
}

func BenchmarkVariableWithByteMatch(b *testing.B) {
	a := assert.New(b)

	str := "i {{.name}} dont {{.action}}"
	extra := []byte(`{"action":"future","name":"san"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := variableWithByteMatch(str, extra)
		a.NoError(err)
	}
}

func variableWithByteMatch(message string, extra []byte) (newMessage string, err error) {
	idx := strings.Index(message, "{{.")
	if idx < 0 {
		return message, nil
	}

	mm := make(map[string]string, 2)
	if len(extra) > 0 {
		if err = util.JsonUnmarshal(extra, &mm); err != nil {
			return "", err
		}
	}

	msgLen := len(message)

	buff := bytes.NewBuffer(make([]byte, 0, msgLen))
	buff.WriteString(message[:idx])

	for idx < msgLen {
		start := idx + 3
		if start >= msgLen { // start >= msgLen
			return "", fmt.Errorf("error msg len <= start, please check template")
		}
		end := strings.Index(message[start:], "}}")
		if end < 0 {
			return "", fmt.Errorf("error msg end idx < 0, please check template")
		}
		end = start + end
		if end+2 > msgLen { // end > msgLen
			return "", fmt.Errorf("error msg len <= end, please check template")
		}
		key := message[start:end]
		if v, ok := mm[key]; ok {
			buff.WriteString(v)
		} else {
			buff.WriteString(message[idx : end+2])
		}

		idx = strings.Index(message[end+2:], "{{.")
		if idx < 0 {
			buff.WriteString(message[end+2:]) // write remain
			break
		} else {
			idx += end + 2
			buff.WriteString(message[end+2 : idx])
		}
	}

	newMessage = buff.String()
	return
}

func BenchmarkVariableWithFastTemplate(b *testing.B) {
	a := assert.New(b)

	str := "i {{.name}} dont {{.action}}"
	extra := []byte(`{"action":"future","name":"san"}`)

	t := fasttemplate.New(str, "{{.", "}}")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := variableWithFastTemplate(t, extra)
		a.NoError(err)
	}
}

func variableWithFastTemplate(template *fasttemplate.Template, extra []byte) (newMessage string, err error) {
	mm := make(map[string]interface{}, 2)
	if len(extra) > 0 {
		if err = util.JsonUnmarshal(extra, &mm); err != nil {
			return "", err
		}
	}

	newMessage = template.ExecuteString(mm)
	return
}

func TestFastTemplate(t *testing.T) {
	a := assert.New(t)

	str := "i {{.name}} dont {{.action}}"
	extra := []byte(`{"action":"future","name":"san"}`)
	msg, err := variableWithFastTemplate(fasttemplate.New(str, "{{.", "}}"), extra)
	a.NoError(err)
	a.Equal("i san dont future", msg)
}

func BenchmarkFmt(b *testing.B) {
	var code int64 = 23456
	var tag int64 = 75646
	service := "open-contract-core"
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%d:%d:%s", code, tag, service)
	}
}

func BenchmarkFmt1(b *testing.B) {
	var code int64 = 23456
	var tag int64 = 75646
	service := "open-contract-core"
	for i := 0; i < b.N; i++ {
		_ = cast.Int64toa(code) + ":" + cast.Int64toa(tag) + ":" + service
	}
}

func TestLang(t *testing.T) {
	convey.Convey("ETL lang", t, func() {
		raw := "en-US"
		convey.So(strings.ToLower(raw), convey.ShouldEqual, "en-us")
	})
}
