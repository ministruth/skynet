package page

import (
	"html/template"
	"reflect"
	"strconv"
	"time"

	"github.com/hako/durafmt"
	"github.com/inhies/go-bytesize"
	"github.com/ztrue/tracerr"
)

var defaultFunc = template.FuncMap{
	"time":    formatTime,
	"since":   sinceTime,
	"bytes":   formatBytes,
	"percent": formatPercent,
	"dict":    templateDict,
}

var (
	ErrInvalidDictCall  = tracerr.New("invalid dict call")
	ErrInvalidDictValue = tracerr.New("dict values must be maps")
	ErrNonArrayValue    = tracerr.New("specify the key for non array values")
)

func templateDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values) == 0 {
		return nil, ErrInvalidDictCall
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
				return nil, ErrInvalidDictValue
			}
		} else {
			i++
			if i == len(values) {
				return nil, ErrNonArrayValue
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
	d := durafmt.Parse(time.Since(t)).LimitFirstN(1)
	return d.String()
}

func formatBytes(n uint64) string {
	b := bytesize.New(float64(n))
	return b.String()
}

func formatPercent(p float64, c int) string {
	return strconv.FormatFloat(p, 'f', c, 64)
}
