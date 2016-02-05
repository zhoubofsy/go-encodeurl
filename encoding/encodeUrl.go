package encurl

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type Func func(obj interface{}) (string, bool, error)

var (
	funcs = make(map[string]Func)
	lock  = sync.RWMutex{}
)

func init() {
	AddEncodeFunc(ifStringIsNotEmpty)
}

func Translate(obj interface{}) (url.Values, []error) {
	if reflect.TypeOf(obj).Kind() != reflect.Struct &&
		reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return nil, []error{errors.New("obj must be a struct or pointer")}
	}
	var errs []error
	values := url.Values{}
	e := reflect.TypeOf(obj).Elem()

	for i := 0; i < e.NumField(); i++ {
		field := e.Field(i)
		structFieldValue := reflect.ValueOf(obj).Elem().FieldByName(field.Name)
		if structFieldValue.IsValid() {
			tab := strings.Split(field.Tag.Get("url"), ",")
			if len(tab) > 1 {
				lock.RLock()
				if validator, ok := funcs[tab[1]]; ok {
					val, ok, err := validator(structFieldValue.Interface())
					if err != nil {
						errs = append(errs, err)
					} else if ok {
						values.Add(tab[0], val)
					}
				} else {
					errs = append(errs, fmt.Errorf("%v doesn't exist", tab[1]))
				}
				lock.RUnlock()
			} else {
				errs = append(errs, fmt.Errorf("No method for %v(%v) field", field.Name, tab[0]))
			}
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return values, nil
}

func AddEncodeFunc(fnct ...Func) (errs []error) {
	lock.Lock()

	errs = make([]error, 1)
	for _, f := range fnct {
		tab := strings.Split(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), ".")
		if len(tab) > 1 {
			name := tab[len(tab)-1]
			funcs[name] = f
			if _, ok := funcs[name]; ok {
				errs = append(errs, fmt.Errorf("%v already exist", name))
			}
			funcs[name] = f
		}
	}
	lock.Unlock()
	return
}

func PrintAllFunctions(out io.Writer) {
	lock.RLock()
	for k := range funcs {
		fmt.Fprintf(out, "%v\n", k)
	}
	lock.RUnlock()
}
