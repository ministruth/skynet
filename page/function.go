package page

import (
	"errors"
	"fmt"
	"html/template"
	"math"
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
	"split":   splitPage,
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

func apiVersion(in string) string {
	return api.APIVERSION + in
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

type pagination struct {
	Text     string
	Number   int
	Active   bool
	Disabled bool
}

func splitPage(p int, n int) []*pagination {
	var ret []*pagination

	ret = append(ret, &pagination{
		Text:     "«",
		Number:   int(math.Max(1, float64(p-1))),
		Disabled: p == 1,
	})
	if n <= 5 {
		for i := 1; i <= n; i++ {
			ret = append(ret, &pagination{
				Text:   fmt.Sprint(i),
				Number: i,
				Active: i == p,
			})
		}
	} else {
		low := int(math.Max(float64(p-2), 1))
		high := int(math.Min(float64(p+2), float64(n)))
		if low == 1 {
			for i := 1; i <= 4; i++ {
				ret = append(ret, &pagination{
					Text:   fmt.Sprint(i),
					Number: i,
					Active: i == p,
				})
			}
			ret = append(ret, &pagination{
				Text:     "...",
				Number:   0,
				Disabled: true,
			})
			ret = append(ret, &pagination{
				Text:   fmt.Sprint(n),
				Number: n,
			})
		} else if high == n {
			ret = append(ret, &pagination{
				Text:   "1",
				Number: 1,
			})
			ret = append(ret, &pagination{
				Text:     "...",
				Number:   0,
				Disabled: true,
			})
			for i := n - 3; i <= n; i++ {
				ret = append(ret, &pagination{
					Text:   fmt.Sprint(i),
					Number: i,
					Active: i == p,
				})
			}
		} else {
			ret = append(ret, &pagination{
				Text:   "1",
				Number: 1,
			})
			ret = append(ret, &pagination{
				Text:     "...",
				Number:   0,
				Disabled: true,
			})
			ret = append(ret, &pagination{
				Text:   fmt.Sprint(p - 1),
				Number: p - 1,
			})
			ret = append(ret, &pagination{
				Text:   fmt.Sprint(p),
				Number: p,
				Active: true,
			})
			ret = append(ret, &pagination{
				Text:   fmt.Sprint(p + 1),
				Number: p + 1,
			})
			ret = append(ret, &pagination{
				Text:     "...",
				Number:   0,
				Disabled: true,
			})
			ret = append(ret, &pagination{
				Text:   fmt.Sprint(n),
				Number: n,
			})
		}
	}
	ret = append(ret, &pagination{
		Text:     "»",
		Number:   int(math.Min(float64(p+1), float64(n))),
		Disabled: p == n,
	})
	return ret
}
