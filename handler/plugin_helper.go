package handler

import (
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/ztrue/tracerr"
)

type pluginHelper struct{}

func (*pluginHelper) Eval(str string) (string, error) {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	v, err := i.Eval(str)
	if err != nil {
		return "", tracerr.Wrap(err)
	}
	return v.Interface().(string), nil
}
