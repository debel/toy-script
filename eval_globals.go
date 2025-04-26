package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
)

func injectBuiltins(f *frame) {
	f.set("@map", toyMap)
	f.set("@get", toyGet)
	f.set("@set", toySet)
	f.set("@has", toyHas)
	f.set("@len", toyLen)
	f.set("@close", toyClose)
	f.set("@await", toyAwait)
	f.set("@collect", toyCollect)
	f.set("=", toyEqual)

	// aliases
	f.set("@push", toySet)
	f.set("@pull", toyGet)
}

func httpGet(a ...any) any {
	url := a[0].(string)
	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Sprintf("http.get: failed to get %s: %s", url, err.Error()))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http.get: failed to read response body for %s: %s", url, err.Error()))
	}

	return body
}

func jsonParse(a ...any) any {
	data := a[0].([]byte)
	var parsed any
	err := json.Unmarshal(data, &parsed)
	if err != nil {
		panic(fmt.Sprintf("json.parse: %s", err.Error()))
	}

	return parsed
}

func stdioPrint(v ...any) any {
	fmt.Println(v...)
	return nil
}

func stdioRead(_ ...any) any {
	reader := bufio.NewReader(os.Stdin)
	message, _ := reader.ReadString('\n')
	return strings.Trim(message, " \n\r\t")
}

func stdioBuildStr(v ...any) any {
	str := strings.Builder{}
	for _, vi := range v {
		if vi != nil {
			str.WriteString(vi.(string))
		}
	}

	return str.String()
}

func toyMap(a ...any) any {
	fn := a[0].(funcType)
	switch obj := a[1].(type) {
	case []any:
		results := []any{}
		for _, el := range obj {
			results = append(results, fn(el))
		}

		return results
	case map[string]any:
		results := map[string]any{}
		for k, v := range obj {
			results[k] = fn(v)
		}

		return results
	case chan any:
		results := make(chan any)
		for el := range obj {
			results <- fn(el)
		}
		close(results)

		return results
	}

	panic(fmt.Sprintf("unable to map over %v", a[1]))
}

func toyGet(a ...any) any {
	switch obj := a[0].(type) {
	case []any:
		idx := a[1].(int)
		return obj[idx]
	case map[string]any:
		key := a[1].(string)
		return obj[key]
	case chan any:
		return <-obj
	}

	panic(fmt.Sprintf("unsupported collection for get: %v", a[0]))
}

func toyHas(a ...any) any {
	query := a[1]
	switch obj := a[0].(type) {
	case []any:
		return slices.Contains(obj, query)
	case map[string]any:
		key := query.(string)
		_, ok := obj[key]
		return ok
	case chan any:
		for el := range obj {
			if el == query {
				return true
			}
		}

		return false
	}

	panic(fmt.Sprintf("unsupported collection for has: %v", a[0]))
}

func toySet(a ...any) any {
	switch obj := a[0].(type) {
	case []any:
		idx, isIndex := a[1].(int)
		if isIndex && len(a) > 2 {
			obj[idx] = a[2]
		} else {
			obj[len(obj)] = a[1]
		}

		return nil
	case map[string]any:
		key := a[1].(string)
		obj[key] = a[2]

		return nil
	case chan any:
		select {
		case obj <- a[1]:
		default:
		}
		return nil
	}

	panic(fmt.Sprintf("unsupported collection for set: %v %s", a[0], reflect.TypeOf(a[0])))
}

func toyLen(a ...any) any {
	switch obj := a[0].(type) {
	case string:
		return len(obj)
	case []any:
		return len(obj)
	case map[string]any:
		return len(obj)
	case chan any:
		return len(obj)
	}

	panic(fmt.Sprintf("unsupported collection for len: %v", a[0]))
}

func toyEqual(a ...any) any {
	last := a[0]
	for _, ai := range a[1:] {
		if ai != last {
			return false
		}
	}

	return true
}

func toyClose(a ...any) any {
	close(a[0].(chan any))
	return nil
}

func toyAwait(a ...any) any {
	return <-a[0].(chan any)
}

func toyCollect(a ...any) any {
	switch collection := a[0].(type) {
	case map[string]any:
		return maps.Values(collection)
	case chan any:
		result := []any{}
		for el := range collection {
			result = append(result, el)
		}

		return result
	default:
		return a
	}
}
