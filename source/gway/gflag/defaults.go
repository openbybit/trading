package gflag

import (
	"flag"

	"github.com/iancoleman/strcase"
)

// ParseFlags 扩展Parse,默认使用小驼峰解析,错误默认使用ContinueOnError方式
func ParseFlags(args []string, from interface{}, options ...FillerOption) error {
	defaultOpts := []FillerOption{WithFieldRenamer(strcase.ToLowerCamel)}
	options = append(defaultOpts, options...)
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	fillter := New(options...)
	if err := fillter.Fill(flagset, from); err != nil {
		return err
	}

	if err := flagset.Parse(args); err != nil {
		return err
	}

	return nil
}
