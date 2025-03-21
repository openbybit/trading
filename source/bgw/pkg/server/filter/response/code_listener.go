package response

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"text/template"

	"code.bydev.io/fbu/gateway/gway.git/galert"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/gcore/observer"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/valyala/fasttemplate"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/util"
	"bgw/pkg/config_center"
	"bgw/pkg/config_center/nacos"
)

type msgSourceType int64

const (
	MsgSource_Unknown = iota
	MsgSource_OnlyCode
	MsgSource_BackEnd
	MsgSource_FrontEnd
	MsgSource_OnlyMessage
)

const (
	ErrorMetaPrefix  = "bgw-errors-meta-"
	ErrorMatchPrefix = "bgw-errors-match-"
)

const (
	codeListenerAlertTitle = "错误码模版更新"
	logMessageHit          = "codeMessage hit"
	logVarErr              = "Variable error"
	logKeyTempMsg          = "template-msg"
)

var (
	codeLoaders     = make(map[string]*CodeListener)
	MessageVariable = template.New("message_variable").Option("missingkey=zero")

	codelock sync.RWMutex
)

type ErrorMetas struct {
	App   string         `yaml:"app"`
	Codes []*CodeMessage `yaml:"codes"`
}

type ErrorMathcs struct {
	App           string          `yaml:"app"`
	Matchs        []*CodeMatch    `yaml:"matchs"`
	SpecialMatchs []*SpeCodeMatch `yaml:"special_matchs"`
}

type SpeCodeMatch struct {
	SourceCode int64 `yaml:"source_code"`
}

type CodeMatch struct {
	SourceCodeList []int64 `yaml:"source_code_list"`
	SourceCode     int64   `yaml:"source_code"`
	TargetCode     int64   `yaml:"target_code"`
	Tag            int64   `yaml:"tag"`
	Service        string  `yaml:"service"`
}

type CodeMessage struct {
	Code int64             `yaml:"code"`
	Lang map[string]string `yaml:"lang"`
	Tag  int64             `yaml:"tag"`
	lang map[string]*MessageTemplate
}

// GetMessage get message template
func (l *CodeMessage) GetMessage(lang string) *MessageTemplate {
	// 'en-US' -> 'en-us'
	language := strings.ToLower(lang)
	if m, ok := l.lang[language]; ok {
		return m
	}
	if v, ok := l.lang["en-us"]; ok {
		return v
	}
	return l.lang["en"]
}

type CodeListener struct {
	ctx           context.Context
	nacosCfg      config_center.Configure
	codes         map[string]*CodeMessage // code int64 -> CodeMessage
	matchs        map[string]*CodeMatch   // code int64 -> CodeMatch
	specialMatchs map[int64]*SpeCodeMatch // code -> config
}

// codeLoader get code loader
func codeLoader(app string) (*CodeListener, bool) {
	codelock.RLock()
	defer codelock.RUnlock()

	if v, ok := codeLoaders[app]; ok {
		return v, true
	}
	return nil, false
}

// setCodeLoader set code loader
func setCodeLoader(app string) {
	if app == "" {
		return
	}

	if _, ok := codeLoader(app); ok {
		return
	}

	codenc, err := nacos.NewNacosConfigure(
		context.Background(),
		nacos.WithGroup(constant.BGW_GROUP),              // specified group
		nacos.WithNameSpace(constant.BGWConfigNamespace), // namespace isolation
	)
	if err != nil {
		glog.Error(context.Background(), "CodeListener listen NewNacosConfigure error", glog.String("error", err.Error()))
		panic(err)
	}

	c := &CodeListener{
		ctx:      context.Background(),
		nacosCfg: codenc,
	}

	file := ErrorMetaPrefix + app
	if err := c.nacosCfg.Listen(c.ctx, file, c); err != nil {
		msg := fmt.Sprintf("CodeListener listen error, err = %s, file = %s", err.Error(), file)
		galert.Error(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
		return
	}
	matchFile := ErrorMatchPrefix + app
	if err := c.nacosCfg.Listen(c.ctx, matchFile, c); err != nil {
		msg := fmt.Sprintf("CodeListener match listen error, err = %s, file = %s", err.Error(), matchFile)
		galert.Error(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
		return
	}

	codelock.Lock()
	defer codelock.Unlock()
	codeLoaders[app] = c

	glog.Info(c.ctx, "set code loader", glog.String("app", app), glog.String("file", file))
}

// OnEvent code on event handler
func (c *CodeListener) OnEvent(event observer.Event) error {
	e, ok := event.(*observer.DefaultEvent)
	if !ok {
		return nil
	}
	if e.Value == "" {
		return nil
	}
	glog.Info(context.TODO(), "CodeListener OnEvent", glog.String("key", e.Key), glog.String("content", util.ToMD5(e.Value)))

	if strings.Contains(e.Key, ErrorMetaPrefix) {
		metas := &ErrorMetas{}
		if err := util.YamlUnmarshalString(e.Value, metas); err != nil {
			msg := fmt.Sprintf("CodeListener staticPreCheck error, err = %s, EventKey = %s", err.Error(), e.Key)
			galert.Error(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
			return nil
		}
		if metas.App == "" || !strings.HasSuffix(e.Key, metas.App) {
			msg := fmt.Sprintf("CodeListener app is invalid, EventKey = %s", e.Key)
			galert.Error(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
			return nil
		}

		c.buildMetas(metas)
		msg := fmt.Sprintf("CodeListener build success, EventKey = %s", e.Key)
		galert.Info(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
	} else if strings.Contains(e.Key, ErrorMatchPrefix) {
		metas := &ErrorMathcs{}
		if err := util.YamlUnmarshalString(e.Value, metas); err != nil {
			msg := fmt.Sprintf("CodeListener staticPreCheck error, err = %s, EventKey = %s", err.Error(), e.Key)
			galert.Error(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
			return nil
		}
		if metas.App == "" || !strings.HasSuffix(e.Key, metas.App) {
			msg := fmt.Sprintf("CodeListener app is invalid, EventKey = %s", e.Key)
			galert.Error(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
			return nil
		}

		c.buildMatchs(metas)
		msg := fmt.Sprintf("CodeListener build success, EventKey = %s", e.Key)
		galert.Info(c.ctx, msg, galert.WithTitle(codeListenerAlertTitle))
	} else {
		return fmt.Errorf("CodeListener unknown file: %s", e.Key)
	}

	return nil
}

// formatCode format code
func (c *CodeListener) formatCode(code, tag int64) string {
	return cast.Int64toa(code) + ":" + cast.Int64toa(tag)
}

// formatMatch format match
func (c *CodeListener) formatMatch(code, tag int64, service string) string {
	return cast.Int64toa(code) + ":" + cast.Int64toa(tag) + ":" + service
}

func (c *CodeListener) buildMetas(e *ErrorMetas) {
	var (
		tmpCodes = make(map[string]*CodeMessage, 100)
	)

	for _, v := range e.Codes {
		// build template
		msgTemplate := make(map[string]*MessageTemplate, 100)
		for lang, msg := range v.Lang {
			tm := &MessageTemplate{msg: msg}
			if idx := strings.Index(msg, "{{."); idx >= 0 {
				tm.template = fasttemplate.New(msg, "{{.", "}}")
			}
			msgTemplate[lang] = tm
		}
		v.lang = msgTemplate

		tmpCodes[c.formatCode(v.Code, v.Tag)] = v
	}
	codelock.Lock()
	defer codelock.Unlock()
	c.codes = tmpCodes
}

func (c *CodeListener) buildMatchs(e *ErrorMathcs) {
	var (
		tmpMatchs        = make(map[string]*CodeMatch, 100)
		tmpSpecialMatchs = make(map[int64]*SpeCodeMatch, 10)
	)

	for _, v := range e.Matchs {
		// Prefer using SourceCodeList
		if len(v.SourceCodeList) == 0 {
			tmpMatchs[c.formatMatch(v.SourceCode, v.Tag, v.Service)] = v
		}
		for _, vv := range v.SourceCodeList {
			tmpMatchs[c.formatMatch(vv, v.Tag, v.Service)] = v
		}
	}
	for _, v := range e.SpecialMatchs {
		tmpSpecialMatchs[v.SourceCode] = v
	}

	codelock.Lock()
	defer codelock.Unlock()
	c.matchs = tmpMatchs
	c.specialMatchs = tmpSpecialMatchs
}

type MessageTemplate struct {
	template *fasttemplate.Template
	msg      string
}

func (c *CodeListener) codeMessage(lang string, code, tag int64) *MessageTemplate {
	codelock.RLock()
	defer codelock.RUnlock()

	if v, ok := c.codes[c.formatCode(code, 0)]; ok {
		return v.GetMessage(lang)
	}
	return nil
}

func (c *CodeListener) codeMatch(code, tag int64, service string) *CodeMatch {
	codelock.RLock()
	defer codelock.RUnlock()

	_, ok := c.specialMatchs[code]
	if tag > 0 && ok {
		if v, ok := c.matchs[c.formatMatch(code, tag, service)]; ok {
			return v
		}
		// If there is no match for service, then get again without service
		if v, ok := c.matchs[c.formatMatch(code, tag, "")]; ok {
			return v
		}
		return nil
	}
	if v, ok := c.matchs[c.formatMatch(code, 0, service)]; ok {
		return v
	}
	if v, ok := c.matchs[c.formatMatch(code, 0, "")]; ok {
		return v
	}
	return nil
}

// parseCodeMessage parse code and message
func (c *CodeListener) parseCodeMessage(ctx context.Context, extra []byte, tag int64, lang string,
	code int64, message string, source msgSourceType, service string) (int64, string, bool) {
	// If the source is not configured, do not parse code or message
	if source == 0 {
		return code, message, false
	}
	glog.Debug(ctx, "parseCodeMessage", glog.Int64("source", int64(source)),
		glog.String("service", service),
		glog.Int64("tag", tag))

	if source == MsgSource_OnlyMessage {
		m := c.codeMessage(lang, code, tag)

		if m == nil {
			return code, message, false
		}
		glog.Debug(ctx, logMessageHit, glog.Int64("tag", tag), glog.Int64("code", code))

		msg, err := c.Variable(m, extra)
		if err != nil {
			glog.Error(ctx, logVarErr, glog.Int64("tag", tag), glog.Int64("code", code),
				glog.String(logKeyTempMsg, m.msg), glog.String("message", message),
				glog.String("lang", lang), glog.Int64("source", int64(source)), glog.String("error", err.Error()))
			return code, message, false
		}
		return code, msg, true
	}

	targetCode := code
	match := c.codeMatch(code, tag, service)
	if match != nil {
		glog.Debug(ctx, "hit match", glog.Any("match", match))
		targetCode = match.TargetCode
	}

	// A/B -> C
	switch source {
	case MsgSource_OnlyCode: // only code
		return targetCode, message, false
	case MsgSource_BackEnd: // code and A/B message
		if m := c.codeMessage(lang, code, tag); m != nil {
			glog.Debug(ctx, logMessageHit, glog.Int64("tag", tag), glog.Int64("code", code))
			msg, err := c.Variable(m, extra)
			if err == nil {
				return targetCode, msg, true
			}
			glog.Error(ctx, logVarErr, glog.Int64("tag", tag), glog.Int64("code", code),
				glog.String(logKeyTempMsg, m.msg), glog.String("message", message),
				glog.String("lang", lang), glog.Int64("source", int64(source)), glog.String("error", err.Error()))
		}
		return targetCode, message, false
	case MsgSource_FrontEnd: // code and C message
		if m := c.codeMessage(lang, targetCode, tag); m != nil {
			glog.Debug(ctx, logMessageHit, glog.Int64("tag", tag), glog.Int64("code", targetCode))
			msg, err := c.Variable(m, extra)
			if err == nil {
				return targetCode, msg, true
			}
			glog.Error(ctx, logVarErr, glog.Int64("tag", tag), glog.Int64("code", targetCode),
				glog.String(logKeyTempMsg, m.msg), glog.String("message", message), glog.String("lang", lang),
				glog.Int64("source", int64(source)), glog.String("error", err.Error()))
		}
		return targetCode, message, false
	}
	return code, message, false
}

// Variable handle variable
func (c *CodeListener) Variable(t *MessageTemplate, extra []byte) (string, error) {
	if t.template == nil {
		return t.msg, nil
	}

	mm := make(map[string]interface{}, 2)
	if len(extra) > 0 {
		if err := util.JsonUnmarshal(extra, &mm); err != nil {
			return "", err
		}
	}

	return t.template.ExecuteString(mm), nil
}

// GetEventType get event type
func (c *CodeListener) GetEventType() reflect.Type {
	return nil
}

// GetPriority get priority
func (c *CodeListener) GetPriority() int {
	return 0
}
