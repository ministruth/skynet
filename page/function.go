package page

import (
	"errors"
	"html/template"
	"reflect"
	"strconv"
	"time"

	"github.com/hako/durafmt"
	"github.com/inhies/go-bytesize"
)

var defaultFunc = template.FuncMap{
	// TODO: Add page functions
	"time":    formatTime,
	"since":   sinceTime,
	"bytes":   formatBytes,
	"percent": formatPercent,
	"dict":    templateDict,
}

var (
	InvalidDictCallError = errors.New("Invalid dict call")
	DictValueError       = errors.New("Dict values must be maps")
	NonArrayValueError   = errors.New("Specify the key for non array values")
)

func templateDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values) == 0 {
		return nil, InvalidDictCallError
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
				return nil, DictValueError
			}
		} else {
			i++
			if i == len(values) {
				return nil, NonArrayValueError
			}
			dict[key] = values[i]
		}

	}
	return dict, nil
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func sinceTime(t time.Time) string {
	d := durafmt.Parse(time.Now().Sub(t)).LimitFirstN(1)
	return d.String()
}

func formatBytes(n uint64) string {
	b := bytesize.New(float64(n))
	return b.String()
}

func formatPercent(p float64, c int) string {
	return strconv.FormatFloat(p, 'f', c, 64)
}
