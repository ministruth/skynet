package page

import (
	"errors"
	"html/template"
	"reflect"
	"skynet/api"
	"strconv"
	"time"

	"github.com/hako/durafmt"
	"github.com/inhies/go-bytesize"
)

var defaultFunc = template.FuncMap{
	// TODO: Add page functions
	"api":     apiVersion,
	"time":    formatTime,
	"since":   sinceTime,
	"bytes":   formatBytes,
	"percent": formatPercent,
	"dict":    templateDict,
}

func templateDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values) == 0 {
		return nil, errors.New("invalid dict call")
	}

	dict := make(map[string]interface{})

	for i := 0; i < len(values); i++ {
		key, isset := values[i].(string)
		if !isset {
			if reflect.TypeOf(values[i]).Kind() == reflect.Map {
				m := values[i].(map[string]interface{})
				for i, v := range m {
					dict[i] = v
				}
			} else {
				return nil, errors.New("dict values must be maps")
			}
		} else {
			i++
			if i == len(values) {
				return nil, errors.New("specify the key for non array values")
			}
			dict[key] = values[i]
		}

	}
	return dict, nil
}

func apiVersion(in string) (string, error) {
	return api.APIVERSION + in, nil
}

func formatTime(t time.Time) (string, error) {
	return t.Format("2006-01-02 15:04:05"), nil
}

func sinceTime(t time.Time) (string, error) {
	d := durafmt.Parse(time.Now().Sub(t)).LimitFirstN(1)
	return d.String(), nil
}

func formatBytes(n uint64) (string, error) {
	b := bytesize.New(float64(n))
	return b.String(), nil
}

func formatPercent(p float64, c int) (string, error) {
	return strconv.FormatFloat(p, 'f', c, 64), nil
}
