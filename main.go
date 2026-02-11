package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"bytes"
	"encoding/json"
	"bufio"
	"os"
	"reflect"
	"io"
	"time"
	"math"
	"runtime"
	"sort"
	"unsafe"
	"sync"
	"path/filepath"
	"net/http"
	"mime/multipart"
	"github.com/gin-gonic/gin"
	gotdotenv "github.com/joho/godotenv"
)

var wincreatetime float64 = OSLcastNumber(time.Now().UnixMilli())
var system_os = runtime.GOOS

// This is a set of funtions that are used in the compiler for OSL.go

func OSLlen(s any) int {
	if s == nil {
		return 0
	}
	switch s := s.(type) {
	case string:
		return len(s)
	case []any:
		return len(s)
	case []string:
		return len(s)
	case []int:
		return len(s)
	case []float64:
		return len(s)
	case []bool:
		return len(s)
	case []byte:
		return len(s)
	case []io.Reader:
		return len(s)
	}
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		return v.Len()
	}
	if v.Kind() == reflect.Map {
		return v.Len()
	}
	if v.Kind() == reflect.String {
		return len(v.String())
	}
	panic("OSLlen, invalid type: " + v.Kind().String())
}

func OSLcastString(s any) string {
	switch s := s.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	case []any:
		return JsonStringify(s)
	case map[string]any, map[string]string, map[string]int, map[string]float64, map[string]bool:
		return JsonStringify(s)
	case io.Reader:
		data, err := io.ReadAll(s)
		if err != nil {
			panic("OSLcastString: failed to read io.Reader:" + err.Error())
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", s)
	}
}

func OSLcastObject(s any) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	switch s := s.(type) {
	case map[string]any:
		return s
	default:
		panic("OSLcastObject: invalid type, " + reflect.TypeOf(s).String())
	}
}

func OSLcastArray(values ...any) []any {
	if len(values) == 1 {
		v := values[0]

		if arr, ok := v.([]any); ok {
			return arr
		}

		rv := reflect.ValueOf(v)

		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return []any{}
			}
			rv = rv.Elem()
		}

		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			out := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				out[i] = rv.Index(i).Interface()
			}
			return out
		}

		return []any{v}
	}

	return values
}

func OSLequal(a any, b any) bool {
	if a == b {
		return true
	}
	return strings.EqualFold(OSLcastString(a), OSLcastString(b))
}

func OSLnotEqual(a any, b any) bool {
	if a == b {
		return false
	}
	return !strings.EqualFold(OSLcastString(a), OSLcastString(b))
}

func OSLcastInt(i any) int {
	if i == nil {
		return 0
	}
	switch i := i.(type) {
	case string:
		f, _ := strconv.ParseFloat(string(i), 64)
		return int(f)
	case int:
		return i
	case float64:
		return int(i)
	case bool:
		if i {
			return 1
		}
		return 0
	case int8:
		return int(i)
	case int16:
		return int(i)
	case int32:
		return int(i)
	case int64:
		return int(i)
	case json.Number:
		f, _ := i.Float64()
		return int(f)
	default:
		panic("OSLcastInt, invalid type: " + reflect.TypeOf(i).String())
	}
}

func OSLlogValues(values ...any) {
	for _, v := range values {
		OSLlog(v)
	}
}

func OSLlog(v any) {
	if v == nil {
		fmt.Println("null")
	}
	switch v := v.(type) {
	case map[string]any:
		fmt.Println(JsonStringify(v))
		return
	case []any:
		fmt.Println(JsonStringify(v))
		return
	case string, int, float64, bool:
		fmt.Println(v)
		return
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			fmt.Println("null")
			return
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		fmt.Println(JsonStringify(OSLcastArray(v)))
		return
	}

	if rv.Kind() == reflect.Map {
		fmt.Println(JsonStringify(OSLcastObject(v)))
		return
	}

	fmt.Println(v)
}

func OSLisFunc(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Func
}

func OSLcallFunc(fn any, self any, params []any) any {
	if fn == nil {
		return nil
	}

	if params == nil {
		params = []any{}
	}

	if self != nil {
		params = append([]any{self}, params...)
	}

	rv := reflect.ValueOf(fn)
	if rv.Kind() != reflect.Func {
		panic("OSLcallFunc: invalid type: " + reflect.TypeOf(fn).String())
	}

	ft := rv.Type()
	numIn := ft.NumIn()

	isVariadic := ft.IsVariadic()

	args := make([]reflect.Value, 0, len(params))

	for i := range params {
		var pt reflect.Type

		if isVariadic && i >= numIn-1 {
			pt = ft.In(numIn - 1).Elem()
		} else {
			pt = ft.In(i)
		}

		var av reflect.Value

		if params[i] == nil {
			switch pt.Kind() {
			case reflect.Interface, reflect.Pointer, reflect.Map,
				reflect.Slice, reflect.Func, reflect.Chan:
				av = reflect.Zero(pt)
			default:
				panic("OSLcallFunc: nil is not assignable to " + pt.String())
			}
		} else {
			av = reflect.ValueOf(params[i])

			at := av.Type()

			if at.AssignableTo(pt) {
			} else if at.ConvertibleTo(pt) {
				av = av.Convert(pt)
			} else if pt.Kind() == reflect.Interface && at.Implements(pt) {
			} else {
				panic(
					"OSLcallFunc: cannot use " + at.String() +
						" as " + pt.String(),
				)
			}
		}

		args = append(args, av)
	}

	out := rv.Call(args)

	switch len(out) {
	case 0:
		return nil
	case 1:
		return out[0].Interface()
	default:
		res := make([]any, len(out))
		for i := range out {
			res[i] = out[i].Interface()
		}
		return res
	}
}

func OSLsort(arr []any) []any {
	if arr == nil {
		return nil
	}

	sort.Slice(arr, func(i, j int) bool {
		return OSLcastString(arr[i]) < OSLcastString(arr[j])
	})
	return arr
}

func OSLsortBy(arr []any, key any) []any {
	if arr == nil {
		return nil
	}

	if OSLisFunc(key) {
		sort.Slice(arr, func(i, j int) bool {
			ki := OSLcallFunc(key, nil, []any{arr[i]})
			kj := OSLcallFunc(key, nil, []any{arr[j]})

			return OSLless(ki, kj)
		})
		return arr
	}

	keyStr := OSLcastString(key)
	sort.Slice(arr, func(i, j int) bool {
		ai, ok1 := arr[i].(map[string]any)
		aj, ok2 := arr[j].(map[string]any)

		if !ok1 || !ok2 {
			return false
		}

		ki := ai[keyStr]
		kj := aj[keyStr]

		return OSLless(ki, kj)
	})

	return arr
}

func OSLless(a any, b any) bool {
	if a == b {
		return false
	}
	return OSLcastString(a) < OSLcastString(b)
}

func OSLgreater(a any, b any) bool {
	if a == b {
		return false
	}
	return OSLcastString(a) > OSLcastString(b)
}

func OSLcastNumber(n any) float64 {
	if n == nil {
		return 0
	}
	switch n := n.(type) {
	case string:
		f, _ := strconv.ParseFloat(string(n), 64)
		return f
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case bool:
		if n {
			return float64(1)
		}
		return float64(0)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return float64(n.(float64))
	}
}

func OSLcastBool(b any) bool {
	if b == nil {
		return false
	}

	switch b := b.(type) {
	case string:
		return len(b) > 0
	case int:
		return b == 1
	case bool:
		return b
	case []any:
		return len(b) > 0
	case map[string]any:
		return len(b) > 0
	default:
		v := reflect.ValueOf(b)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			return OSLcastBool(v.Elem().Interface())
		}
		panic("OSLcastBool, invalid type: " + v.Kind().String())
	}
}

func OSLcastUsable(s any) any {
	switch s := s.(type) {
	case string, int, bool, float64, map[string]any:
		return s
	case []any:
		result := make([]any, len(s))
		for i, v := range s {
			result[i] = OSLcastUsable(v)
		}
		return result
	default:
		rv := reflect.ValueOf(s)
		if rv.Kind() == reflect.Slice {
			result := make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				result[i] = OSLcastUsable(rv.Index(i).Interface())
			}
			return result
		}
		return fmt.Sprintf("%v", s)
	}
}

func OSLrandom[T int | float64](low, high T) T {
	if high <= low {
		return low
	}

	switch any(low).(type) {
	case int:
		return T(rand.Intn(int(high-low)) + int(low))

	case float64:
		return (T(rand.Float64()) * (high - low)) + low
	}

	panic("OSLrandom: unsupported type")
}

func OSLnullishCoaless(a any, b any) any {
	if a == nil {
		return b
	}
	return a
}

func OSLSplit(s string, sep string) []any {
	split := strings.Split(s, sep)
	out := make([]any, len(split))
	for i, v := range split {
		out[i] = v
	}
	return out
}

func JsonStringify(obj any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(obj); err != nil {
		return ""
	}
	return strings.TrimRight(buf.String(), "\n")
}

func JsonParse(str string) any {
	if strings.TrimSpace(str) == "" {
		return interface{}(nil)
	}

	var obj any
	decoder := json.NewDecoder(strings.NewReader(str))
	decoder.UseNumber()
	if err := decoder.Decode(&obj); err != nil {
		return interface{}(nil)
	}
	return obj
}

func JsonFormat(obj any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		return ""
	}
	return strings.TrimRight(buf.String(), "\n")
}

// Math operation wrappers for OSL behavior

func input(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func OSLgetItem(a any, b any) any {
	if a == nil {
		return nil
	}

	if sm, ok := a.(*SafeMap[string, any]); ok {
		val, _ := sm.Get(OSLcastString(b))
		return val
	}

	if v, ok := a.(map[string]any); ok {
		return v[OSLcastString(b)]
	}

	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	key := OSLcastString(b)

	switch v.Kind() {
	case reflect.Map:
		mk := reflect.ValueOf(key)
		val := v.MapIndex(mk)
		if val.IsValid() {
			return val.Interface()
		}
	case reflect.Slice, reflect.Array:
		idx := OSLcastInt(b) - 1 // OSL 1-indexed
		if idx < 0 || idx >= v.Len() {
			return nil
		}
		return v.Index(idx).Interface()
	case reflect.Struct:
		// Try exact field name
		field := v.FieldByName(key)
		if field.IsValid() && field.CanInterface() {
			return field.Interface()
		}
		// Optionally: loop through fields and match lowercase names
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if strings.EqualFold(f.Name, key) && v.Field(i).CanInterface() {
				return v.Field(i).Interface()
			}
		}
	case reflect.String:
		idx := OSLcastInt(b) - 1
		s := v.String()
		if idx < 0 || idx >= len(s) {
			return ""
		}
		return string(s[idx])
	default:
		panic("OSLgetItem: invalid type (" + v.Kind().String() + ")")
	}

	return nil
}

func OSLjoin[T string | []any, T2 string | []any](a T, b T2) T {
	switch aSlice := any(a).(type) {
	case []any:
		switch bVal := any(b).(type) {
		case []any:
			return any(append(aSlice, bVal...)).(T)
		}
	}

	return any(OSLcastString(a) + OSLcastString(b)).(T)
}

func OSLadd[T float64 | int](a T, b T) T {
	return T(OSLcastNumber(a) + OSLcastNumber(b))
}

func OSLsub[T float64 | int](a T, b T) T {
	return T(OSLcastNumber(a) - OSLcastNumber(b))
}

func OSLmultiply[BT float64 | int](a any, b BT) any {
	if str, ok := a.(string); ok {
		n := OSLcastNumber(b)
		if n < 0 {
			return ""
		}
		return strings.Repeat(str, int(n))
	}

	return OSLcastNumber(a) * OSLcastNumber(b)
}

func OSLdivide[T float64 | int](a T, b T) T {
	return T(OSLcastNumber(a) / OSLcastNumber(b))
}

func OSLmod[T float64 | int](a T, b T) T {
	return T(math.Mod(OSLcastNumber(a), OSLcastNumber(b)))
}

func OSLmin[T float64 | int](a T, b T) T {
	if a < b {
		return a
	}
	return b
}

func OSLmax[T float64 | int](a T, b T) T {
	if a > b {
		return a
	}
	return b
}

func OSLround(n any) int {
	if n == nil {
		return 0
	}
	switch n := n.(type) {
	case int:
		return n
	case float64:
		return int(n + 0.5)
	default:
		panic("OSLround, invalid type: " + reflect.TypeOf(n).String())
	}
}

func OSLceil(n any) float64 {
	switch n := n.(type) {
	case int:
		return float64(n)
	case float64:
		return math.Ceil(n)
	default:
		panic("OSLceil, invalid type: " + reflect.TypeOf(n).String())
	}
}

func OSLfloor(n any) float64 {
	switch n := n.(type) {
	case int:
		return float64(n)
	case float64:
		return math.Floor(n)
	default:
		panic("OSLfloor, invalid type: " + reflect.TypeOf(n).String())
	}
}

func OSLtrim(s any, from int, to int) string {
	str := []rune(OSLcastString(s))

	start := from - 1
	end := to

	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = len(str) + end + 1
	}

	if start > len(str) {
		start = len(str)
	}
	if end > len(str) {
		end = len(str)
	}

	if start > end {
		start, end = end, start
	}

	return string(str[start:end])
}

func OSLwait(seconds float64) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

func OSLslice(s any, start int, end int) []any {
	arr := OSLcastArray(s)
	n := len(arr)

	start = start - 1
	if start < 0 {
		start = 0
	} else if start > n {
		start = n
	}

	if end < 0 {
		end = n + end + 1
	}
	if end > n {
		end = n
	} else if end < 0 {
		end = 0
	}

	if start > end {
		start, end = end, start
	}

	return arr[start:end]
}

func OSLpadStart(s string, length int, pad string) string {
	if len(s) >= length {
		return s
	}
	return strings.Repeat(pad, length-len(s)) + s
}

func OSLpadEnd(s string, length int, pad string) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(pad, length-len(s))
}

func OSLtypeof(s any) string {
	switch s.(type) {
	case string:
		return "string"
	case int:
		return "int"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "any"
	}
}

func OSLKeyIn(b any, a any) bool {
	if a == nil {
		return false
	}

	key := OSLcastString(b)
	if sm, ok := a.(*SafeMap[string, any]); ok {
		_, exists := sm.Get(key)
		return exists
	}

	switch a := a.(type) {
	case map[string]any:
		_, ok := a[key]
		return ok
	case []any:
		for _, v := range a {
			if OSLcastString(v) == key {
				return true
			}
		}
		return false
	}

	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Map:
		mapKeyType := v.Type().Key()
		mk := reflect.ValueOf(key)
		if !mk.Type().AssignableTo(mapKeyType) {
			if mapKeyType.Kind() == reflect.String {
				mk = reflect.ValueOf(key)
			} else {
				return false
			}
		}
		val := v.MapIndex(mk)
		return val.IsValid()

	case reflect.Slice, reflect.Array:
		idx := OSLcastInt(b) - 1
		return idx >= 0 && idx < v.Len()

	case reflect.String:
		idx := OSLcastInt(b) - 1
		return idx >= 0 && idx < len(v.String())

	case reflect.Struct:
		if field := v.FieldByName(key); field.IsValid() {
			return true
		}
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if strings.EqualFold(f.Name, key) {
				return true
			}
		}
		return false

	default:
		return false
	}
}

func OSLdelete(a any, b any) any {
	if a == nil {
		return nil
	}

	if sm, ok := a.(*SafeMap[string, any]); ok {
		sm.Delete(OSLcastString(b))
		return a
	}

	switch a := a.(type) {
	case map[string]any:
		delete(a, OSLcastString(b))
		return a
	case []any:
		idx := OSLcastInt(b) - 1
		if idx < 0 || idx >= len(a) {
			return a
		}
		return append(a[:idx], a[idx+1:]...)
	}

	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	key := OSLcastString(b)

	switch v.Kind() {
	case reflect.Map:
		mk := reflect.ValueOf(key)
		if mk.Type().AssignableTo(v.Type().Key()) {
			v.SetMapIndex(mk, reflect.Value{})
		}
		return v.Interface()

	case reflect.Slice:
		idx := OSLcastInt(b) - 1
		if idx < 0 || idx >= v.Len() {
			return v.Interface()
		}
		newSlice := reflect.AppendSlice(v.Slice(0, idx), v.Slice(idx+1, v.Len()))
		return newSlice.Interface()

	default:
		return a
	}
}

func OSLsetItem(a any, b any, value any) bool {
	if a == nil {
		return false
	}

	if sm, ok := a.(*SafeMap[string, any]); ok {
		sm.Set(OSLcastString(b), value)
		return true
	}

	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	key := OSLcastString(b)

	switch v.Kind() {
	case reflect.Map:
		mk := reflect.ValueOf(key)
		if !mk.IsValid() {
			return false
		}

		var mv reflect.Value
		if value == nil {
			mv = reflect.Zero(v.Type().Elem())
		} else {
			mv = reflect.ValueOf(value)
		}

		if mk.Type().AssignableTo(v.Type().Key()) && mv.Type().AssignableTo(v.Type().Elem()) {
			v.SetMapIndex(mk, mv)
			return true
		}
		return false

	case reflect.Slice:
		idx := OSLcastInt(b) - 1
		if idx < 0 || idx >= v.Len() {
			return false
		}
		elem := reflect.ValueOf(value)
		if elem.Type().AssignableTo(v.Index(idx).Type()) {
			v.Index(idx).Set(elem)
			return true
		}
		return false

	case reflect.Struct:
		field := v.FieldByName(key)
		if !field.IsValid() {
			return false
		}

		var val reflect.Value
		if value == nil {
			val = reflect.Zero(field.Type())
		} else {
			val = reflect.ValueOf(value)
		}

		return setFieldUnsafe(field, val)
	}

	return false
}

func setFieldUnsafe(field reflect.Value, val reflect.Value) bool {
	if !field.CanAddr() {
		return false
	}

	if !val.Type().AssignableTo(field.Type()) {
		if val.Type().ConvertibleTo(field.Type()) {
			val = val.Convert(field.Type())
		} else {
			return false
		}
	}

	ptr := unsafe.Pointer(field.UnsafeAddr())
	reflect.NewAt(field.Type(), ptr).Elem().Set(val)
	return true
}

func OSLarrayJoin(a any, b any) string {
	var out strings.Builder
	sep := OSLcastString(b)
	arr := OSLcastArray(a)

	for _, v := range arr {
		out.WriteString(OSLcastString(v) + sep)
	}

	return strings.TrimSuffix(out.String(), sep)
}

func OSLgetKeys(a any) []any {
	if sm, ok := a.(*SafeMap[string, any]); ok {
		keys := sm.Keys()
		result := make([]any, len(keys))
		for i, k := range keys {
			result[i] = k
		}
		return result
	}

	switch a := a.(type) {
	case map[string]any:
		keys := make([]any, len(a))
		i := 0
		for k := range a {
			keys[i] = k
			i++
		}
		return keys
	case []any:
		keys := make([]any, len(a))
		for i := range a {
			keys[i] = i
		}
		return keys
	default:
		return []any{}
	}
}

func OSLgetValues(a any) []any {
	if sm, ok := a.(*SafeMap[string, any]); ok {
		values := sm.Values()
		result := make([]any, len(values))
		for i, v := range values {
			result[i] = v
		}
		return result
	}

	switch a := a.(type) {
	case map[string]any:
		values := make([]any, len(a))
		i := 0
		for _, v := range a {
			values[i] = v
			i++
		}
		return values
	case []any:
		values := make([]any, len(a))
		i := 0
		for _, v := range a {
			values[i] = v
			i++
		}
		return values
	default:
		return []any{}
	}
}

func OSLcontains(a any, b any) bool {
	if sm, ok := a.(*SafeMap[string, any]); ok {
		_, exists := sm.Get(OSLcastString(b))
		return exists
	}

	switch a := a.(type) {
	case map[string]any:
		_, ok := a[OSLcastString(b)]
		return ok
	case []any:
		for _, v := range a {
			if OSLcastString(v) == OSLcastString(b) {
				return true
			}
		}
		return false
	case string:
		return strings.Contains(a, OSLcastString(b))
	default:
		return false
	}
}

func OSLappend(a *[]any, b any) []any {
	*a = append(*a, b)
	return *a
}

func OSLpop(a *[]any) any {
	if len(*a) == 0 {
		return nil
	}
	last := (*a)[len(*a)-1]
	*a = (*a)[:len(*a)-1]
	return last
}

func OSLshift(a *[]any) any {
	if len(*a) == 0 {
		return nil
	}
	first := (*a)[0]
	*a = append([]any{}, (*a)[1:]...)
	return first
}

func OSLprepend(a *[]any, b any) []any {
	*a = append([]any{b}, *a...)
	return *a
}

func OSLclone(a any) any {
	switch a := a.(type) {
	case map[string]any:
		b := make(map[string]any, len(a))
		for k, v := range a {
			b[k] = OSLclone(v)
		}
		return b
	case []any:
		b := make([]any, len(a))
		for i, v := range a {
			b[i] = OSLclone(v)
		}
		return b
	default:
		return a
	}
}

// worker handling

var OSLself any = nil

func OSLworker(props map[string]any) map[string]any {
	props["createdTime"] = time.Now()
	props["processTime"] = 0
	props["alive"] = true
	props["kill"] = func() {
		props["alive"] = false
	}
	go (func() {
		OSLself = props
		OSLcallFunc(props["oncreate"], props, nil)
		for {
			startTime := time.Now()
			OSLself = props
			OSLcallFunc(props["onframe"], props, nil)
			props["processTime"] = OSLcastNumber(props["processTime"]) + time.Since(startTime).Seconds()
			if props["alive"] != true {
				props["alive"] = false
				OSLself = props
				OSLcallFunc(props["onkill"], props, nil)
				break
			}
		}
	})()
	return props
}

type SafeMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewSafeMap[K comparable, V any](defaults map[K]V) *SafeMap[K, V] {
	sm := &SafeMap[K, V]{
		data: make(map[K]V, len(defaults)),
	}
	for k, v := range defaults {
		sm.data[k] = v
	}
	return sm
}

func (m *SafeMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value // regular map syntax here
}

func (m *SafeMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.data[key] // regular map syntax here
	return value, ok
}

func (m *SafeMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *SafeMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func (m *SafeMap[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]V, 0, len(m.data))
	for _, v := range m.data {
		values = append(values, v)
	}
	return values
}

// name: requests
// description: HTTP utilities
// author: Mist
// requires: net/http, encoding/json, io

type HTTP struct {
	Client *http.Client
}

func extractHeadersAndBody(data map[string]any) (headers map[string]string, body io.Reader) {
	headers = make(map[string]string)
	if data != nil {
		if raw, ok := data["body"]; ok {
			switch v := raw.(type) {
			case string:
				body = bytes.NewReader([]byte(v))
			case []byte:
				body = bytes.NewReader(v)
			case map[string]any:
				body = bytes.NewReader([]byte(JsonStringify(v)))
				headers["Content-Type"] = "application/json"
			default:
				buf, _ := json.Marshal(v)
				body = bytes.NewReader(buf)
				headers["Content-Type"] = "application/json"
			}
		}

		for k, v := range data {
			if k == "body" {
				continue
			}
			headers[k] = OSLcastString(v)
		}
	}
	return headers, body
}

func (h *HTTP) doRequest(method, url string, data map[string]any) map[string]any {
	headers, body := extractHeadersAndBody(data)

	out := make(map[string]any)
	out["headers"] = nil
	out["body"] = nil
	out["raw"] = nil
	out["status"] = 0
	out["success"] = false

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return out
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()

	respHeaders := make(map[string]any)
	for k, v := range resp.Header {
		if len(v) == 1 {
			respHeaders[k] = v[0]
		} else {
			respHeaders[k] = v
		}
	}

	out["status"] = resp.StatusCode
	out["headers"] = respHeaders
	out["raw"] = resp
	out["success"] = true

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return out
	}

	out["body"] = respBody

	return out
}

func (h *HTTP) Get(url any, data ...map[string]any) map[string]any {
	var m map[string]any
	if len(data) > 0 {
		m = data[0]
	}
	return h.doRequest(http.MethodGet, OSLcastString(url), m)
}

func (h *HTTP) Post(url any, data map[string]any) map[string]any {
	return h.doRequest(http.MethodPost, OSLcastString(url), data)
}

func (h *HTTP) Put(url any, data map[string]any) map[string]any {
	return h.doRequest(http.MethodPut, OSLcastString(url), data)
}

func (h *HTTP) Patch(url any, data map[string]any) map[string]any {
	return h.doRequest(http.MethodPatch, OSLcastString(url), data)
}

func (h *HTTP) Delete(url any, data ...map[string]any) map[string]any {
	var m map[string]any
	if len(data) > 0 {
		m = data[0]
	}
	return h.doRequest(http.MethodDelete, OSLcastString(url), m)
}

func (h *HTTP) Options(url any, data ...map[string]any) map[string]any {
	var m map[string]any
	if len(data) > 0 {
		m = data[0]
	}
	return h.doRequest(http.MethodOptions, OSLcastString(url), m)
}

func (h *HTTP) Head(url any, data ...map[string]any) map[string]any {
	var m map[string]any
	if len(data) > 0 {
		m = data[0]
	}
	out := map[string]any{"success": false}
	headers, _ := extractHeadersAndBody(m)
	req, err := http.NewRequest(http.MethodHead, OSLcastString(url), nil)
	if err != nil {
		return out
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return out
	}
	out["status"] = resp.StatusCode
	defer resp.Body.Close()
	out["success"] = true
	return out
}

var requests = &HTTP{Client: http.DefaultClient}
// name: fs
// description: File system utilities
// author: Mist
// requires: os, path/filepath

type FS struct{}

func (FS) ReadFile(path any) string {
	data, err := os.ReadFile(OSLcastString(path))
	if err != nil {
		return ""
	}
	return string(data)
}

func (FS) ReadFileBytes(path any) []byte {
	data, err := os.ReadFile(OSLcastString(path))
	if err != nil {
		return []byte{}
	}
	return data
}

func (FS) WriteFile(path any, data any) bool {
	err := os.WriteFile(OSLcastString(path), []byte(OSLcastString(data)), 0644)
	return err == nil
}

func (FS) Rename(oldPath any, newPath any) bool {
	err := os.Rename(OSLcastString(oldPath), OSLcastString(newPath))
	return err == nil
}

func (FS) Exists(path any) bool {
	_, err := os.Stat(OSLcastString(path))
	return err == nil
}

func (FS) Remove(path any) bool {
	pathStr, ok := path.(string)
	if !ok || pathStr == "" {
		return false
	}

	info, err := os.Stat(pathStr)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return os.RemoveAll(pathStr) == nil
	}

	return os.Remove(pathStr) == nil
}

func (FS) Mkdir(path any) bool {
	err := os.Mkdir(OSLcastString(path), 0755)
	return err == nil
}

func (FS) MkdirAll(path any) bool {
	err := os.MkdirAll(OSLcastString(path), 0755)
	return err == nil
}

func (FS) CopyDir(srcPath any, dstPath any) bool {
	src := OSLcastString(srcPath)
	dst := OSLcastString(dstPath)

	entries, err := os.ReadDir(src)
	if err != nil {
		return false
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return false
	}

	for _, entry := range entries {
		srcFile := filepath.Join(src, entry.Name())
		dstFile := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			ok := (FS{}).CopyDir(srcFile, dstFile)
			if !ok {
				return false
			}
			continue
		}

		in, err := os.Open(srcFile)
		if err != nil {
			return false
		}

		out, err := os.Create(dstFile)
		if err != nil {
			in.Close()
			return false
		}

		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return false
		}

		in.Close()
		out.Close()

		if info, err := os.Stat(srcFile); err == nil {
			_ = os.Chmod(dstFile, info.Mode())
		}
	}

	return true
}

func (FS) ReadDir(path any) []any {
	files, err := os.ReadDir(OSLcastString(path))
	if err != nil {
		return []any{}
	}
	names := make([]any, len(files))
	for i, f := range files {
		names[i] = f.Name()
	}
	return names
}

func (FS) ReadDirAll(path any) []map[string]any {
	dir := OSLcastString(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []map[string]any{}
	}

	filesOut := make([]map[string]any, len(entries))
	for i, f := range entries {
		filesOut[i] = map[string]any{
			"name":  f.Name(),
			"ext":   filepath.Ext(f.Name()),
			"path":  filepath.Join(dir, f.Name()),
			"isDir": f.IsDir(),
			"type":  f.Type(),
		}
	}

	return filesOut
}

func (FS) WalkDir(path any, fn func(path string, file map[string]any, control map[string]any)) {
	dir := OSLcastString(path)
	filepath.WalkDir(dir, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		fileData := map[string]any{
			"name":    entry.Name(),
			"ext":     filepath.Ext(entry.Name()),
			"path":    p,
			"isDir":   entry.IsDir(),
			"size":    info.Size(),
			"mode":    info.Mode(),
			"modTime": info.ModTime(),
			"sys":     info.Sys(),
			"type":    entry.Type(),
		}

		control := map[string]any{
			"skip": false,
		}
		fn(p, fileData, control)
		if control["skip"] == true {
			return filepath.SkipDir
		}
		return nil
	})
}

func (FS) IsDir(path any) bool {
	info, err := os.Stat(OSLcastString(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (FS) Getwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func (FS) Chdir(path any) bool {
	err := os.Chdir(OSLcastString(path))
	return err == nil
}

func (FS) JoinPath(path ...any) string {
	stringPath := make([]string, len(path))
	for i, p := range path {
		stringPath[i] = OSLcastString(p)
	}
	return filepath.Join(stringPath...)
}

func (FS) GetBase(path any) string {
	return filepath.Base(OSLcastString(path))
}

func (FS) GetDir(path any) string {
	return filepath.Dir(OSLcastString(path))
}

func (FS) GetExt(path any) string {
	return filepath.Ext(OSLcastString(path))
}

func (FS) GetParts(path any) []any {
	stringPath := OSLcastString(path)
	return []any{filepath.Base(stringPath), filepath.Dir(stringPath), filepath.Ext(stringPath)}
}

func (FS) GetSize(path any) float64 {
	info, err := os.Stat(OSLcastString(path))
	if err != nil {
		return 0
	}
	return float64(info.Size())
}

func (FS) GetModTime(path any) float64 {
	info, err := os.Stat(OSLcastString(path))
	if err != nil {
		return 0.0
	}
	return float64(info.ModTime().UnixMilli())
}

func (FS) GetStat(path any) map[string]any {
	info, err := os.Stat(OSLcastString(path))
	if err != nil {
		return map[string]any{"success": false}
	}
	return map[string]any{
		"success": true,
		"name":    filepath.Base(info.Name()),
		"ext":     filepath.Ext(info.Name()),
		"path":    info.Name(),
		"isDir":   info.IsDir(),
		"size":    info.Size(),
		"mode":    info.Mode(),
		"modTime": info.ModTime().UnixMicro(),
		"sys":     info.Sys(),
	}
}

func (FS) EvalSymlinks(path any) string {
	pathStr := OSLcastString(path)
	absPath, err := filepath.EvalSymlinks(pathStr)
	if err != nil {
		return ""
	}
	return absPath
}

// Global instance
var fs = FS{}
func up(c *gin.Context) {
  c.String(200, "ok")
}

func handleDevAuth(c *gin.Context) {
  var token string = OSLcastString(c.Query("v"))
  if OSLequal(token, "") {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "missing token",
    })
    return
  }
  var resp map[string]any = OSLcastObject(requests.Get(("https://api.rotur.dev/validate?key=rotur-app-store&v=" + token)))
  if (OSLgetItem(resp, "success") != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "failed to validate token",
    })
    return
  }
  var json map[string]any = OSLcastObject(JsonParse(OSLcastString(OSLgetItem(resp, "body"))))
  if OSLnotEqual(OSLgetItem(json, "error"), nil) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": OSLgetItem(json, "error"),
    })
    return
  }
  if (OSLgetItem(json, "valid") != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "invalidate token",
    })
    return
  }
  var sessionId string = OSLcastString(randomString(32))
  var userId string = OSLcastString(OSLgetItem(json, "id"))
  var username string = OSLcastString(OSLgetItem(json, "username"))
  OSLsetItem(userIdToUsername, userId, username)
  OSLsetItem(sessions, sessionId, userId)
  fs.WriteFile("./sessions.json", JsonFormat(sessions))
  c.SetCookie("session_id", sessionId, 604800, "/", "", false, true)
  c.JSON(200, map[string]any{
    "ok": true,
    "token": token,
  })
}
func handleLogout(c *gin.Context) {
  c.SetCookie("session_id", "", 3600, "/", "", false, true)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleMyApps(c *gin.Context) {
  var userId string = OSLcastString(c.MustGet("userId"))
  var apps []any = OSLcastArray(getAppsByDev(userId))
  c.JSON(200, apps)
}
func handleCreateApp(c *gin.Context) {
  var appname string = OSLcastString(c.Query("appname"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var username string = OSLcastString(getUsernameFromUserId(userId))
  if OSLequal(appname, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing appname",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", strings.ToLower(appname)))
  if fs.Exists(path) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app already exists",
    })
    return
  }
  var adminToken = os.Getenv("ADMIN_TOKEN")
  if OSLequal(adminToken, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing admin token",
    })
    return
  }
  var resp = requests.Post((("https://api.rotur.dev/admin/transfer_credits?amount=9&from=" + username) + "&to=rotur"), map[string]any{
    "authorization": adminToken,
  })
  if OSLequal(OSLgetItem(resp, "success"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "failed to transfer credits",
    })
    return
  }
  var body string = OSLcastString(OSLgetItem(resp, "body"))
  var data map[string]any = OSLcastObject(JsonParse(body))
  if OSLtypeof(data) != "object" {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "invalid response body",
    })
    return
  }
  if OSLnotEqual(OSLgetItem(data, "error"), nil) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": OSLgetItem(data, "error"),
    })
    return
  }
  fs.CopyDir(fs.JoinPath("./apps", ".template"), fs.JoinPath("./apps", strings.ToLower(appname)))
  var info map[string]any = OSLcastObject(JsonParse(fs.ReadFile(fs.JoinPath(path, "info.json.hidden"))))
  OSLsetItem(info, "title", appname)
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  OSLappend(&(authors), userId)
  OSLsetItem(info, "authors", authors)
  fs.WriteFile(fs.JoinPath(path, "info.json.hidden"), JsonFormat(info))
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleDeleteApp(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var userId string = OSLcastString(c.MustGet("userId"))
  if OSLequal(appname, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing appname",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname))
  if (fs.Exists(path) != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  var authors []any = OSLcastArray(OSLgetItem(OSLgetItem(app, "info"), "authors"))
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      fs.Remove(path)
      c.JSON(200, map[string]any{
        "ok": true,
      })
      return
    }
  }
  c.JSON(401, map[string]any{
    "ok": false,
    "error": "not authorized",
  })
}
func handleUpdateApp(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  if OSLequal(appname, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing appname",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname))
  if (fs.Exists(path) != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var infoPath string = OSLcastString(fs.JoinPath(path, "info.json"))
  if (fs.Exists(infoPath) != true) {
    infoPath = fs.JoinPath(path, "info.json.hidden")
    if (fs.Exists(infoPath) != true) {
      c.JSON(400, map[string]any{
        "ok": false,
        "error": "app does not exist",
      })
      return
    }
  }
  var info map[string]any = OSLcastObject(JsonParse(fs.ReadFile(infoPath)))
  var newInfo map[string]any = OSLcastObject(JsonParse(OSLcastString(OSLgetItem(OSLgetItem(c, "Request"), "Body"))))
  var keys []any = OSLgetKeys(newInfo)
  for i := 1; i <= OSLlen(keys); i++ {
    var key string = OSLcastString(OSLgetItem(keys, i))
    if OSLequal(key, "authors") || OSLequal(key, "approved") {
      continue
    }
    if OSLcontains(info, key) {
      OSLsetItem(info, key, OSLgetItem(newInfo, key))
    }
  }
  OSLlogValues(JsonStringify(keys), JsonFormat(info), JsonFormat(newInfo))
  fs.WriteFile(infoPath, JsonFormat(info))
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func updateAppInfo(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname))
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var infoPath string = ""
  if OSLequal(OSLgetItem(app, "hidden"), true) {
    infoPath = fs.JoinPath(path, "info.json.hidden")
  } else {
    infoPath = fs.JoinPath(path, "info.json")
  }
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var newInfo map[string]any = OSLcastObject(JsonParse(OSLcastString(OSLgetItem(OSLgetItem(c, "Request"), "Body"))))
  var currentInfo map[string]any = OSLcastObject(JsonParse(fs.ReadFile(infoPath)))
  var keys []any = OSLgetKeys(newInfo)
  for i := 1; i <= OSLlen(keys); i++ {
    var key string = OSLcastString(OSLgetItem(keys, i))
    if OSLcontains([]any{
      "title",
      "downloads",
      "views",
    }, key) {
      continue
    }
    OSLsetItem(currentInfo, key, OSLgetItem(newInfo, key))
  }
  fs.WriteFile(infoPath, JsonFormat(currentInfo))
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppUpload(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var filetype string = OSLcastString(c.Param("filetype"))
  var path string = OSLcastString(fs.JoinPath("./apps", appname, ("script." + strings.ToLower(filetype))))
  c.File(path)
}
func handleAppDelete(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var filetype string = OSLcastString(c.Param("filetype"))
  var path string = OSLcastString(fs.JoinPath("./apps", appname, ("script." + strings.ToLower(filetype))))
  fs.Remove(path)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppScreenshotUpload(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var screenshot string = OSLcastString(c.Param("screenshot"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var screenshotsPath string = OSLcastString(fs.JoinPath("./apps", appname, "screenshots"))
  if (fs.Exists(screenshotsPath) != true) {
    fs.Mkdir(screenshotsPath)
  }
  var path string = OSLcastString(fs.JoinPath(screenshotsPath, screenshot))
  if fs.Exists(path) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "screenshot already exists",
    })
    return
  }
  var file = OSLcastArray(c.FormFile("file"))
  if OSLnotEqual(OSLgetItem(file, 2), nil) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "invalid file",
    })
    return
  }
  c.SaveUploadedFile(OSLgetItem(file, 1).(*multipart.FileHeader), path)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppScreenshotDelete(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var screenshot string = OSLcastString(c.Param("screenshot"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname, "screenshots", screenshot))
  if (fs.Exists(path) != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "screenshot does not exist",
    })
    return
  }
  fs.Remove(path)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleHideApp(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname))
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      fs.Rename(fs.JoinPath(path, "info.json"), fs.JoinPath(path, "info.json.hidden"))
      c.JSON(200, map[string]any{
        "ok": true,
      })
      return
    }
  }
  c.JSON(401, map[string]any{
    "ok": false,
    "error": "not authorized",
  })
}
func handleShowApp(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname))
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var approved bool = bool(OSLequal(OSLgetItem(readApprovals(), strings.ToLower(appname)), true))
  if (approved != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized, app not approved",
    })
    return
  }
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      fs.Rename(fs.JoinPath(path, "info.json.hidden"), fs.JoinPath(path, "info.json"))
      c.JSON(200, map[string]any{
        "ok": true,
      })
      return
    }
  }
  c.JSON(401, map[string]any{
    "ok": false,
    "error": "not authorized",
  })
}
func handleAppFileList(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var relPath string = OSLcastString(c.DefaultQuery("path", ""))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var dir string = OSLcastString(fs.JoinPath("./apps", appname, relPath))
  if (fs.Exists(dir) != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "invalid path",
    })
    return
  }
  var entries []any = OSLcastArray(fs.ReadDir(dir))
  var files []any = []any{}
  var dirs []any = []any{}
  for i := 1; i <= OSLlen(entries); i++ {
    var f string = OSLcastString(OSLgetItem(entries, i))
    if strings.HasPrefix(f, ".") {
      continue
    }
    if strings.HasPrefix(f, "info.json") {
      continue
    }
    var full string = OSLcastString(fs.JoinPath(dir, f))
    if fs.IsDir(full) {
      OSLappend(&(dirs), f)
    } else {
      OSLappend(&(files), f)
    }
  }
  c.JSON(200, map[string]any{
    "ok": true,
    "path": relPath,
    "dirs": dirs,
    "files": files,
  })
}
func handleAppFileGet(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var file string = OSLcastString(c.Query("file"))
  var relPath string = OSLcastString(c.DefaultQuery("path", ""))
  if OSLequal(file, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing file param",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname, relPath, file))
  var exists bool = bool(fs.Exists(path))
  if (exists != true) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "file not found",
    })
    return
  }
  if strings.HasPrefix(file, "info.json") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "cannot download info.json",
    })
    return
  }
  c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
  c.Header("Pragma", "no-cache")
  c.Header("Expires", "0")
  c.File(path)
}
func handleAppFileSave(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var file string = OSLcastString(c.Query("file"))
  var relPath string = OSLcastString(c.DefaultQuery("path", ""))
  if OSLequal(file, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing file param",
    })
    return
  }
  if strings.HasPrefix(file, "info.json") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "cannot edit info.json directly",
    })
    return
  }
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var authors []any = OSLcastArray(OSLgetItem(OSLgetItem(app, "info"), "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var data string = OSLcastString(OSLgetItem(OSLgetItem(c, "Request"), "Body"))
  var dir string = OSLcastString(fs.JoinPath("./apps", appname, relPath))
  if (fs.Exists(dir) != true) {
    fs.MkdirAll(dir)
  }
  var path string = OSLcastString(fs.JoinPath(dir, file))
  fs.WriteFile(path, data)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppFileDelete(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var file string = OSLcastString(c.Query("file"))
  var relPath string = OSLcastString(c.DefaultQuery("path", ""))
  if OSLequal(file, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing file param",
    })
    return
  }
  if strings.HasPrefix(file, "info.json") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "cannot delete info.json",
    })
    return
  }
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var authors []any = OSLcastArray(OSLgetItem(OSLgetItem(app, "info"), "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var approvals map[string]any = OSLcastObject(readApprovals())
  if (OSLcontains(approvals, strings.ToLower(appname)) != true) {
    OSLdelete(approvals, strings.ToLower(appname))
    writeApprovals(approvals)
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname, relPath, file))
  if (fs.Exists(path) != true) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "file not found",
    })
    return
  }
  fs.Remove(path)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppDirCreate(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var relPath string = OSLcastString(c.DefaultQuery("path", ""))
  var dirName string = OSLcastString(c.DefaultQuery("name", ""))
  if OSLequal(dirName, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing dir name",
    })
    return
  }
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var authors []any = OSLcastArray(OSLgetItem(OSLgetItem(app, "info"), "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var dirPath string = OSLcastString(fs.JoinPath("./apps", appname, relPath, dirName))
  if fs.Exists(dirPath) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "directory already exists",
    })
    return
  }
  fs.MkdirAll(dirPath)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppDirDelete(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var relPath string = OSLcastString(c.DefaultQuery("path", ""))
  if OSLequal(relPath, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing path",
    })
    return
  }
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = OSLcastObject(getAppInfo(appname))
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var authors []any = OSLcastArray(OSLgetItem(OSLgetItem(app, "info"), "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var dirPath string = OSLcastString(fs.JoinPath("./apps", appname, relPath))
  if (fs.Exists(dirPath) != true) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "directory not found",
    })
    return
  }
  if (fs.IsDir(dirPath) != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "not a directory",
    })
    return
  }
  fs.Remove(dirPath)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}

func loadConfig(name string) map[string]any {
  var path string = OSLcastString(fs.JoinPath("./config", name))
  var value = JsonParse(fs.ReadFile(path))
  if OSLtypeof(value) != "object" {
    println((("error: config file '" + name) + "' is not a valid json object"))
  }
  return value.(map[string]any)
}
func loadJson(path string) map[string]any {
  var data string = OSLcastString(fs.ReadFile(path))
  if OSLequal(data, "") {
    return map[string]any{}
  }
  return JsonParse(data).(map[string]any)
}
func getAppInfo(appname string) map[string]any {
  var path string = OSLcastString(fs.JoinPath("./apps", appname))
  var infoPath string = OSLcastString(fs.JoinPath(path, "info.json"))
  var exists bool = bool(fs.Exists(infoPath))
  if (exists != true) {
    infoPath = fs.JoinPath(path, "info.json.hidden")
    exists = fs.Exists(infoPath)
    if (exists != true) {
      return map[string]any{
        "ok": false,
        "error": "app not found",
      }
    }
  }
  var info map[string]any = OSLcastObject(JsonParse(fs.ReadFile(infoPath)))
  OSLsetItem(info, "hidden", strings.HasSuffix(infoPath, ".hidden"))
  var approvals map[string]any = OSLcastObject(readApprovals())
  var approved_status bool = bool(OSLequal(OSLgetItem(approvals, strings.ToLower(appname)), true))
  OSLsetItem(info, "approved", approved_status)
  var iconPath string = OSLcastString(fs.JoinPath(path, "icon.icn"))
  var iconExists bool = bool(fs.Exists(iconPath))
  if iconExists {
    OSLsetItem(info, "icon", fs.ReadFile(iconPath))
  }
  OSLsetItem(info, "supports", []any{})
  var info_supports []any = OSLcastArray(OSLgetItem(info, "supports"))
  OSLsetItem(info, "scripts", map[string]any{})
  var info_scripts map[string]any = OSLcastObject(OSLgetItem(info, "scripts"))
  var fileNames []any = OSLcastArray(fs.ReadDir(path))
  for i := 1; i <= OSLlen(fileNames); i++ {
    var fileName string = OSLcastString(OSLgetItem(fileNames, i))
    if strings.HasPrefix(fileName, ".") {
      continue
    }
    var isDir bool = bool(fs.IsDir(fs.JoinPath(path, fileName)))
    var fileType string = strings.TrimPrefix(fs.GetExt(fileName), ".")
    if OSLnotEqual(OSLgetItem(fileTypes, fileType), nil) {
      var data map[string]any = OSLcastObject(OSLgetItem(fileTypes, fileType))
      var supports []any = OSLcastArray(OSLgetItem(data, "supported"))
      for k := 1; k <= OSLlen(supports); k++ {
        OSLappend(&(info_supports), OSLgetItem(supports, k))
      }
      OSLsetItem(info, "supports", info_supports)
      var scriptPath string = OSLcastString(fs.JoinPath(path, fileName))
      OSLsetItem(info_scripts, fileType, map[string]any{
        "size": fs.GetSize(scriptPath),
      })
    }
    OSLsetItem(fileNames, i, fileName)
    if fileName == "changelogs" && isDir {
      var changelogs []any = OSLcastArray(fs.ReadDir(fs.JoinPath(path, fileName)))
      OSLsetItem(info, "changelogs", []any{})
      for k := 1; k <= OSLlen(changelogs); k++ {
        var cl string = OSLcastString(OSLgetItem(changelogs, k))
        if strings.HasPrefix(cl, ".") {
          continue
        }
        var list []any = OSLcastArray(OSLgetItem(info, "changelogs"))
        list = OSLappend(&(list), cl)
        OSLsetItem(info, "changelogs", list)
      }
    }
    if fileName == "screenshots" && isDir {
      var screenshots []any = OSLcastArray(fs.ReadDir(fs.JoinPath(path, fileName)))
      OSLsetItem(info, "screenshots", []any{})
      for k := 1; k <= OSLlen(screenshots); k++ {
        var screenshot string = OSLcastString(OSLgetItem(screenshots, k))
        if strings.HasPrefix(screenshot, ".") {
          continue
        }
        var screenshots []any = OSLcastArray(OSLgetItem(info, "screenshots"))
        screenshots = OSLappend(&(screenshots), screenshot)
        OSLsetItem(info, "screenshots", screenshots)
      }
    }
  }
  OSLsetItem(info, "author", OSLarrayJoin(OSLgetItem(info, "authors"), ", "))
  var stats map[string]any = OSLcastObject(JsonParse(fs.ReadFile("./stats.json")))
  OSLsetItem(info, "downloads", OSLcastNumber(OSLgetItem(OSLgetItem(stats, "downloads"), strings.ToLower(appname))))
  OSLsetItem(info, "views", OSLcastNumber(OSLgetItem(OSLgetItem(stats, "views"), strings.ToLower(appname))))
  var ownerships map[string]any = OSLcastObject(readOwnerships())
  var owners []any = OSLcastArray(OSLgetItem(ownerships, strings.ToLower(appname)))
  if OSLequal(owners, nil) {
    OSLsetItem(info, "owners", 0)
  } else {
    OSLsetItem(info, "owners", OSLlen(owners))
  }
  return map[string]any{
    "ok": true,
    "info": info,
  }
}
func randomString(length int) string {
  var chars string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
  var result string = ""
  for i := 1; i <= OSLround(length); i++ {
    result = (result + OSLcastString(OSLgetItem(chars, OSLadd(rand.Intn(OSLlen(chars)), 1))))
  }
  return result
}
func getAllApps(showHidden bool) map[string]any {
  var path string = "./apps"
  var exists bool = bool(fs.Exists(path))
  if (exists != true) {
    return map[string]any{
      "ok": false,
      "error": "apps not found",
    }
  }
  var apps map[string]any = map[string]any{}
  var folders []any = OSLcastArray(fs.ReadDir(path))
  for i := 1; i <= OSLlen(folders); i++ {
    var fileName string = OSLcastString(OSLgetItem(folders, i))
    if strings.HasPrefix(fileName, ".") {
      continue
    }
    var info map[string]any = getAppInfo(fileName)
    if OSLequal(OSLgetItem(info, "ok"), true) {
      var app map[string]any = OSLcastObject(OSLgetItem(info, "info"))
      if OSLequal(OSLgetItem(app, "hidden"), false) && OSLequal(OSLgetItem(app, "approved"), true) || OSLequal(showHidden, true) {
        OSLsetItem(apps, fileName, app)
      }
    }
  }
  return map[string]any{
    "ok": true,
    "apps": apps,
  }
}
func getAppsByDev(userId string) []any {
  var resp map[string]any = getAllApps(true)
  if (OSLgetItem(resp, "ok") != true) {
    return []any{}
  }
  var apps map[string]any = OSLcastObject(OSLgetItem(resp, "apps"))
  var appNames []any = OSLgetKeys(apps)
  var myApps []any = []any{}
  for i := 1; i <= OSLlen(appNames); i++ {
    var app map[string]any = OSLcastObject(OSLgetItem(apps, OSLgetItem(appNames, i)))
    var authors []any = OSLcastArray(OSLgetItem(app, "authors"))
    for j := 1; j <= OSLlen(authors); j++ {
      var author string = OSLcastString(OSLgetItem(authors, j))
      if OSLequal(author, userId) {
        OSLappend(&(myApps), app)
      }
    }
  }
  return myApps
}
func getAppFile(appname string, filetype string) map[string]any {
  var path string = OSLcastString(fs.JoinPath("./apps", appname, ("script." + strings.ToLower(filetype))))
  var exists bool = bool(fs.Exists(path))
  if (exists != true) {
    return map[string]any{
      "ok": false,
      "error": "app not found",
    }
  }
  return map[string]any{
    "ok": true,
    "data": fs.ReadFile(path),
  }
}
func readApprovals() map[string]any {
  var path string = OSLcastString(fs.JoinPath("./config", "approvals.json"))
  if (fs.Exists(path) != true) {
    fs.WriteFile(path, "{}")
  }
  var data = JsonParse(fs.ReadFile(path))
  if OSLtypeof(data) != "object" {
    return map[string]any{}
  }
  return data.(map[string]any)
}
func writeApprovals(approvals map[string]any) {
  fs.WriteFile(fs.JoinPath("./config", "approvals.json"), JsonFormat(approvals))
}
func readOwnerships() map[string]any {
  var path string = OSLcastString(fs.JoinPath("./config", "ownership.json"))
  if (fs.Exists(path) != true) {
    fs.WriteFile(path, "{}")
  }
  var data map[string]any = OSLcastObject(JsonParse(fs.ReadFile(path)))
  if OSLtypeof(data) != "object" {
    return map[string]any{}
  }
  return data
}
func writeOwnerships(ownerships map[string]any) {
  fs.WriteFile(fs.JoinPath("./config", "ownership.json"), JsonFormat(ownerships))
}
func getReviews(appname string) []any {
  var reviewPath string = OSLcastString(fs.JoinPath("./reviews", (appname + ".json")))
  if (fs.Exists(reviewPath) != true) {
    return []any{}
  }
  var reviews []any = OSLcastArray(JsonParse(fs.ReadFile(reviewPath)))
  return reviews
}
func getReviewsWithUsernames(appname string) []any {
  var reviews []any = getReviews(appname)
  var reviewsWithUsernames []any = []any{}
  for i := 1; i <= OSLlen(reviews); i++ {
    var review map[string]any = OSLcastObject(OSLgetItem(reviews, i))
    var userId string = OSLcastString(OSLgetItem(review, "user"))
    var username string = getUsernameFromUserId(userId)
    if OSLequal(username, "") {
      username = userId
    }
    OSLappend(&(reviewsWithUsernames), map[string]any{
      "user": username,
      "content": OSLgetItem(review, "content"),
    })
  }
  return reviewsWithUsernames
}
func writeReview(appname string, by_user_id string, content string) bool {
  var reviewPath string = OSLcastString(fs.JoinPath("./reviews", (appname + ".json")))
  if (fs.Exists(reviewPath) != true) {
    fs.WriteFile(reviewPath, "[]")
  }
  var reviews []any = OSLcastArray(JsonParse(fs.ReadFile(reviewPath)))
  OSLappend(&(reviews), map[string]any{
    "user": by_user_id,
    "content": content,
  })
  fs.WriteFile(reviewPath, JsonFormat(reviews))
  return true
}
func setUserIdFromUsername(username string, userId string) {
  OSLsetItem(userIdToUsername, userId, username)
  fs.WriteFile("./userids.json", JsonFormat(userIdToUsername))
}
func getUsernameFromUserId(userId string) string {
  if OSLcontains(userIdToUsername, userId) {
    return OSLcastString(OSLgetItem(userIdToUsername, userId))
  }
  return ""
}
func migrateAuthorNamesToUsernames(userIds []any) []any {
  var usernames []any = []any{}
  for i := 1; i <= OSLlen(userIds); i++ {
    var userId string = OSLcastString(OSLgetItem(userIds, i))
    var username string = getUsernameFromUserId(userId)
    if OSLnotEqual(username, "") {
      OSLappend(&(usernames), username)
    } else {
      OSLappend(&(usernames), userId)
    }
  }
  return usernames
}

var buttons []any = []any{
  map[string]any{
    "Name": "Dashboard",
    "Src": "bar-chart",
    "Link": "/",
  },
  map[string]any{
    "Name": "My Apps",
    "Src": "list",
    "Link": "/apps",
  },
  map[string]any{
    "Name": "Create App",
    "Src": "plus",
    "Link": "/create",
  },
}
func mainObject(c *gin.Context) map[string]any {
  var username string = OSLcastString(c.MustGet("username"))
  var send_buttons []any = OSLcastArray(OSLclone(buttons))
  if OSLcontains([]any{
    "mist",
    "flufi",
    "iris",
  }, strings.ToLower(username)) {
    OSLappend(&(send_buttons), map[string]any{
      "Name": "Admin",
      "Src": "check-circle",
      "Link": "/admin",
    })
  }
  return map[string]any{
    "Username": username,
    "Buttons": send_buttons,
  }
}
func homePage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  var apps []any = getAppsByDev(OSLcastString(OSLgetItem(main, "Username")))
  var numApps int = OSLlen(apps)
  var totalDownloads int = int(0)
  var totalViews int = int(0)
  var hiddenApps int = int(0)
  for i := 1; i <= OSLlen(apps); i++ {
    var app map[string]any = OSLcastObject(OSLgetItem(apps, i))
    if OSLequal(OSLgetItem(app, "hidden"), true) {
      hiddenApps = OSLadd(hiddenApps, 1)
    }
    if OSLnotEqual(OSLgetItem(app, "downloads"), nil) {
      totalDownloads = OSLadd(totalDownloads, OSLcastInt(OSLgetItem(app, "downloads")))
    }
    if OSLnotEqual(OSLgetItem(app, "views"), nil) {
      totalViews = OSLadd(totalViews, OSLcastInt(OSLgetItem(app, "views")))
    }
  }
  var stats map[string]any = map[string]any{
    "NumApps": numApps,
    "TotalDownloads": totalDownloads,
    "TotalViews": totalViews,
    "HiddenApps": hiddenApps,
  }
  c.HTML(200, "index.html", map[string]any{
    "Page": "dashboard",
    "PageName": "Dashboard",
    "Main": main,
    "Stats": stats,
    "Apps": apps,
  })
}
func appsPage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  var apps []any = OSLsortBy(getAppsByDev(OSLcastString(OSLgetItem(main, "Username"))), (func(app any) string {
    return strings.ToLower(OSLcastString(OSLgetItem(app, "title")))
  }))
  c.HTML(200, "index.html", map[string]any{
    "Page": "apps",
    "PageName": "My Apps",
    "Main": main,
    "Apps": apps,
  })
}
func appsInfoPage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  var appname string = strings.ToLower(c.Param("appname"))
  var app map[string]any = getAppInfo(appname)
  var file map[string]any = getAppFile(appname, "osl")
  if OSLequal(OSLgetItem(file, "ok"), true) {
    OSLsetItem(OSLgetItem(app, "info"), "code", OSLcastString(OSLgetItem(file, "data")))
  }
  c.HTML(200, "index.html", map[string]any{
    "Page": "appinfo",
    "PageName": appname,
    "Main": main,
    "App": OSLgetItem(app, "info"),
  })
}
func promotePage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  c.HTML(200, "index.html", map[string]any{
    "Page": "promote",
    "PageName": "Promote",
    "Main": main,
  })
}
func analyticsPage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  var appname string = strings.ToLower(c.Param("appname"))
  var appResp map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(appResp, "ok"), false) {
    c.HTML(404, "index.html", map[string]any{
      "Page": "dashboard",
      "PageName": "Not found",
      "Main": main,
    })
    return
  }
  var app map[string]any = OSLcastObject(OSLgetItem(appResp, "info"))
  var stats map[string]any = map[string]any{
    "Downloads": OSLgetItem(app, "downloads"),
    "Views": OSLgetItem(app, "views"),
    "Hidden": OSLgetItem(app, "hidden"),
    "Version": OSLgetItem(app, "version"),
  }
  c.HTML(200, "index.html", map[string]any{
    "Page": "analytics",
    "PageName": ("Analytics - " + appname),
    "Main": main,
    "Stats": stats,
    "App": app,
  })
}
func createAppPage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  c.HTML(200, "index.html", map[string]any{
    "Page": "create",
    "PageName": "Create App",
    "Main": main,
  })
}
func adminPage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  var resp map[string]any = getAllApps(true)
  if (OSLgetItem(resp, "ok") != true) {
    c.HTML(500, "index.html", map[string]any{
      "Page": "admin",
      "PageName": "Admin",
      "Main": main,
      "Apps": []any{},
    })
    return
  }
  var apps map[string]any = OSLcastObject(OSLgetItem(resp, "apps"))
  var appNames []any = OSLgetKeys(apps)
  var pending []any = []any{}
  var approved []any = []any{}
  for i := 1; i <= OSLlen(appNames); i++ {
    var a map[string]any = OSLcastObject(OSLgetItem(apps, OSLgetItem(appNames, i)))
    if OSLequal(OSLgetItem(a, "approved"), false) {
      OSLappend(&(pending), a)
    } else {
      OSLappend(&(approved), a)
    }
  }
  c.HTML(200, "index.html", map[string]any{
    "Page": "admin",
    "PageName": "Admin",
    "Main": main,
    "Pending": pending,
    "Approved": approved,
  })
}
func publishPage(c *gin.Context) {
  var main map[string]any = mainObject(c)
  c.HTML(200, "index.html", map[string]any{
    "Page": "publish",
    "PageName": "Publish",
    "Main": main,
  })
}
func authPage(c *gin.Context) {
  c.HTML(200, "auth.html", map[string]any{})
}

func handleAppInfo(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var app map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, app)
    return
  }
  OSLsetItem(app, "authors", migrateAuthorNamesToUsernames(OSLcastArray(OSLgetItem(app, "authors"))))
  OSLsetItem(app, "author", OSLarrayJoin(OSLgetItem(app, "authors"), ", "))
  var stats map[string]any = OSLcastObject(JsonParse(fs.ReadFile("./stats.json")))
  var views map[string]any = OSLcastObject(OSLgetItem(stats, "views"))
  OSLsetItem(views, appname, (OSLcastNumber(OSLgetItem(views, appname)) + 1))
  OSLsetItem(stats, "views", views)
  fs.WriteFile("./stats.json", JsonFormat(stats))
  c.JSON(200, app)
}
func handleAppIcon(c *gin.Context) {
  c.File(fs.JoinPath("./apps", strings.ToLower(c.Param("appname")), "icon.icn"))
}
func handleOwnedApps(c *gin.Context) {
  var userId string = OSLcastString(c.MustGet("userId"))
  if OSLequal(userId, "") {
    c.JSON(401, map[string]any{
      "error": "Unauthorized",
    })
    return
  }
  var ownerships map[string]any = OSLcastObject(readOwnerships())
  var appNames []any = OSLgetKeys(ownerships)
  var ownedApps []any = []any{}
  for i := 1; i <= OSLlen(appNames); i++ {
    var name string = OSLcastString(OSLgetItem(appNames, i))
    if OSLcontains(OSLgetItem(ownerships, name), userId) {
      OSLappend(&(ownedApps), name)
    }
  }
  c.JSON(200, map[string]any{
    "ownedApps": OSLsort(ownedApps),
  })
}
func handleAppFile(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var fileType string = OSLcastString(c.Param("filetype"))
  var path string = OSLcastString(fs.JoinPath("./apps", appname, ("script." + strings.ToLower(fileType))))
  var exists bool = bool(fs.Exists(path))
  var app = getAppInfo(appname)
  if (exists != true) || OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  var info = OSLgetItem(app, "info").(map[string]any)
  var userId string = strings.ToLower(OSLcastString(c.MustGet("userId")))
  var token string = OSLcastString(c.Query("auth"))
  var ownerships map[string]any = OSLcastObject(readOwnerships())
  var rawOwners = OSLgetItem(ownerships, appname)
  if OSLequal(rawOwners, nil) {
    rawOwners = []any{}
  }
  var owners []any = OSLcastArray(rawOwners)
  var alreadyOwns bool = false
  if OSLnotEqual(userId, "") {
    alreadyOwns = OSLcontains(owners, strings.ToLower(userId))
  }
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  if OSLequal(OSLlen(authors), 0) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app has no authors",
    })
    return
  }
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      alreadyOwns = true
    }
  }
  if OSLnotEqual(OSLgetItem(info, "price"), nil) && OSLcastNumber(OSLgetItem(info, "price")) > OSLcastNumber(0) && OSLequal(alreadyOwns, false) {
    if OSLequal(token, "") {
      c.JSON(401, map[string]any{
        "ok": false,
        "error": "missing auth token",
      })
      return
    }
    var authorUsername string = getUsernameFromUserId(OSLcastString(OSLgetItem(OSLgetItem(info, "authors"), 1)))
    if OSLequal(authorUsername, "") {
      authorUsername = OSLcastString(OSLgetItem(OSLgetItem(info, "authors"), 1))
    }
    var resp map[string]any = OSLcastObject(requests.Post(("https://api.rotur.dev/me/transfer?auth=" + token), map[string]any{
      "body": map[string]any{
        "amount": OSLcastNumber(OSLgetItem(info, "price")),
        "to": authorUsername,
        "note": (("App store: " + OSLcastString(OSLgetItem(info, "title"))) + " purchase"),
      },
    }))
    if OSLequal(OSLgetItem(resp, "success"), false) || OSLnotEqual(OSLgetItem(resp, "status"), 200) {
      var errorMsg string = OSLcastString(OSLgetItem(JsonParse(OSLcastString(OSLgetItem(resp, "body"))), "error"))
      if OSLcontains(errorMsg, "insufficient funds") {
        errorMsg = "insufficient funds"
      }
      c.JSON(400, map[string]any{
        "ok": false,
        "error": ("failed to purchase: " + errorMsg),
      })
      return
    }
  }
  if OSLnotEqual(userId, "") && OSLequal(alreadyOwns, false) {
    OSLappend(&(owners), strings.ToLower(userId))
    OSLsetItem(ownerships, appname, owners)
    writeOwnerships(ownerships)
  }
  var stats map[string]any = OSLcastObject(JsonParse(fs.ReadFile("./stats.json")))
  var downloads map[string]any = OSLcastObject(OSLgetItem(stats, "downloads"))
  OSLsetItem(downloads, appname, (OSLcastNumber(OSLgetItem(downloads, appname)) + 1))
  OSLsetItem(stats, "downloads", downloads)
  fs.WriteFile("./stats.json", JsonFormat(stats))
  c.File(path)
}
func handleAllApps(c *gin.Context) {
  var token string = OSLcastString(c.Query("auth"))
  var userId string = ""
  OSLlogValues(token)
  if OSLnotEqual(token, "") {
    var parts []any = OSLSplit(token, ",")
    if OSLcastNumber(OSLlen(parts)) >= OSLcastNumber(2) {
      userId = strings.ToLower(OSLcastString(OSLgetItem(parts, 2)))
    }
  }
  var resp map[string]any = getAllApps(false)
  if OSLequal(OSLgetItem(resp, "ok"), false) {
    c.JSON(500, resp)
    return
  }
  if OSLnotEqual(userId, "") {
    var ownerships map[string]any = OSLcastObject(readOwnerships())
    var apps map[string]any = OSLcastObject(OSLgetItem(resp, "apps"))
    var names []any = OSLgetKeys(apps)
    for i := 1; i <= OSLlen(names); i++ {
      var n string = OSLcastString(OSLgetItem(names, i))
      var owners []any = OSLcastArray(OSLgetItem(ownerships, n))
      var owns bool = OSLnotEqual(owners, nil) && OSLcontains(owners, userId)
      OSLsetItem(OSLgetItem(apps, n), "owns", owns)
    }
    OSLsetItem(resp, "apps", apps)
  }
  c.JSON(200, resp)
}
func handleAppScreenshot(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var screenshot string = OSLcastString(c.Param("screenshot"))
  var path string = OSLcastString(fs.JoinPath("./apps", appname, "screenshots", screenshot))
  var exists bool = bool(fs.Exists(path))
  if (exists != true) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  c.File(path)
}
func handleFeatured(c *gin.Context) {
  var featured map[string]any = loadConfig("featured.json")
  if OSLequal(OSLgetItem(featured, "featured"), nil) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "featured app not found",
    })
    return
  }
  c.JSON(200, featured)
}
func handleGetAppReviews(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  c.JSON(200, getReviewsWithUsernames(appname))
}
func handleAppReview(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var review string = OSLcastString(c.Query("content"))
  if OSLequal(review, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "Missing content",
    })
  }
  writeReview(appname, userId, review)
  c.JSON(200, map[string]any{
    "ok": true,
    "error": "Successfully added review",
  })
}

func requireSession(c *gin.Context) {
  var sessionId string = OSLcastString(OSLgetItem(OSLcastArray(c.Cookie("session_id")), 1))
  if OSLequal(sessionId, "") {
    c.Redirect(302, "/auth")
    c.Abort()
    return
  }
  if (OSLcontains(sessions, sessionId) != true) {
    c.Redirect(302, "/auth")
    c.Abort()
    return
  }
  var userId string = OSLcastString(OSLgetItem(sessions, sessionId))
  var username string = getUsernameFromUserId(userId)
  c.Set("userId", userId)
  c.Set("username", username)
  c.Next()
}
var _admins []any = []any{
  "mist",
  "flufi",
  "iris",
}
func requireAdmin(c *gin.Context) {
  var username string = strings.ToLower(OSLcastString(c.MustGet("username")))
  if (OSLcontains(_admins, username) != true) {
    c.JSON(403, map[string]any{
      "ok": false,
      "error": "admin access required",
    })
    c.Abort()
    return
  }
  c.Next()
}
func canAuth(c *gin.Context) {
  var token string = OSLcastString(c.Query("auth"))
  var userId string = ""
  var username string = ""
  if OSLnotEqual(token, "") {
    var resp map[string]any = OSLcastObject(requests.Get(("https://api.rotur.dev/me?auth=" + token)))
    if OSLequal(OSLgetItem(resp, "success"), false) {
      c.JSON(401, map[string]any{
        "ok": false,
        "error": "failed to get user info",
      })
      return
    }
    var user map[string]any = OSLcastObject(JsonParse(OSLcastString(OSLgetItem(resp, "body"))))
    username = strings.ToLower(OSLcastString(OSLgetItem(user, "username")))
    userId = OSLcastString(OSLgetItem(user, "sys.id"))
    setUserIdFromUsername(username, userId)
  }
  c.Set("userId", userId)
  c.Set("username", username)
  c.Next()
}

func handleAdminApprove(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var app map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  var approvals map[string]any = OSLcastObject(readApprovals())
  OSLsetItem(approvals, strings.ToLower(appname), true)
  writeApprovals(approvals)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAdminUnapprove(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var app map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  var approvals map[string]any = OSLcastObject(readApprovals())
  if OSLcontains(approvals, strings.ToLower(appname)) {
    OSLdelete(approvals, strings.ToLower(appname))
    writeApprovals(approvals)
  }
  c.JSON(200, map[string]any{
    "ok": true,
  })
}

func handleAppChangelogUpload(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var changelog string = OSLcastString(c.Param("changelog"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var changelogsPath string = OSLcastString(fs.JoinPath("./apps", appname, "changelogs"))
  if (fs.Exists(changelogsPath) != true) {
    fs.Mkdir(changelogsPath)
  }
  var path string = OSLcastString(fs.JoinPath(changelogsPath, changelog))
  var replace bool = bool(OSLequal(c.Query("replace"), "true"))
  if fs.Exists(path) && (replace != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "changelog already exists",
    })
    return
  }
  var file = OSLcastArray(c.FormFile("file"))
  if OSLnotEqual(OSLgetItem(file, 2), nil) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "invalid file",
    })
    return
  }
  c.SaveUploadedFile(OSLgetItem(file, 1).(*multipart.FileHeader), path)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppChangelogDelete(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var changelog string = OSLcastString(c.Param("changelog"))
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "app does not exist",
    })
    return
  }
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var path string = OSLcastString(fs.JoinPath("./apps", appname, "changelogs", changelog))
  if (fs.Exists(path) != true) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "changelog does not exist",
    })
    return
  }
  fs.Remove(path)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}
func handleAppChangelogRename(c *gin.Context) {
  var appname string = strings.ToLower(c.Param("appname"))
  var oldName string = OSLcastString(c.Query("old"))
  var newName string = OSLcastString(c.Query("new"))
  if OSLequal(oldName, "") || OSLequal(newName, "") {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "missing parameters",
    })
    return
  }
  var userId string = OSLcastString(c.MustGet("userId"))
  var app map[string]any = getAppInfo(appname)
  if OSLequal(OSLgetItem(app, "ok"), false) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "app not found",
    })
    return
  }
  var info map[string]any = OSLcastObject(OSLgetItem(app, "info"))
  var authors []any = OSLcastArray(OSLgetItem(info, "authors"))
  var allowed bool = false
  for i := 1; i <= OSLlen(authors); i++ {
    var author string = OSLcastString(OSLgetItem(authors, i))
    if OSLequal(author, userId) {
      allowed = true
      break
    }
  }
  if (allowed != true) {
    c.JSON(401, map[string]any{
      "ok": false,
      "error": "not authorized",
    })
    return
  }
  var base string = OSLcastString(fs.JoinPath("./apps", appname, "changelogs"))
  var oldPath string = OSLcastString(fs.JoinPath(base, oldName))
  var newPath string = OSLcastString(fs.JoinPath(base, newName))
  if (fs.Exists(oldPath) != true) {
    c.JSON(404, map[string]any{
      "ok": false,
      "error": "old changelog not found",
    })
    return
  }
  if fs.Exists(newPath) {
    c.JSON(400, map[string]any{
      "ok": false,
      "error": "new changelog already exists",
    })
    return
  }
  fs.Rename(oldPath, newPath)
  c.JSON(200, map[string]any{
    "ok": true,
  })
}















var port string = ""
var systems map[string]any = OSLcastObject(loadConfig("systems.json"))
var fileTypes map[string]any = OSLcastObject(loadConfig("filetypes.json"))
var sessions map[string]any = OSLcastObject(loadJson("./sessions.json"))
var userIdToUsername map[string]any = OSLcastObject(loadJson("./userids.json"))
func noCORS(c *gin.Context) {
  c.Header("Access-Control-Allow-Origin", "*")
  c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
  c.Header("Access-Control-Allow-Headers", "*")
  if OSLequal(OSLgetItem(OSLgetItem(c, "Request"), "Method"), "OPTIONS") {
    c.AbortWithStatus(204)
    return
  }
  c.Next()
}
func main() {
  gotdotenv.Load()
  port = os.Getenv("APPSIE_PORT")
  if OSLequal(port, "") {
    port = "5616"
  }
  var r = gin.Default()
  gin.SetMode("release")
  r.LoadHTMLGlob("templates/*")
  r.GET("/admin", requireSession, requireAdmin, adminPage)
  r.GET("/", requireSession, homePage)
  r.GET("/apps", requireSession, appsPage)
  r.GET("/app/:appname", requireSession, appsInfoPage)
  r.GET("/analytics/:appname", requireSession, analyticsPage)
  r.GET("/promote", requireSession, promotePage)
  r.GET("/publish", requireSession, publishPage)
  r.GET("/create", requireSession, createAppPage)
  r.Static("/static", "./static")
  r.GET("/auth", authPage)
  var api = r.Group("/api")
  var apiAdmin = r.Group("/api/admin")
  apiAdmin.Use(requireSession, requireAdmin)
  apiAdmin.POST("/approve/:appname", handleAdminApprove)
  apiAdmin.POST("/unapprove/:appname", handleAdminUnapprove)
  api.GET("/up", up)
  api.GET("/auth", handleDevAuth)
  api.GET("/logout", requireSession, handleLogout)
  api.GET("/apps", requireSession, handleMyApps)
  api.POST("/apps/create", requireSession, handleCreateApp)
  api.DELETE("/apps/delete/:appname", requireSession, handleDeleteApp)
  api.POST("/apps/update/:appname", requireSession, handleUpdateApp)
  api.POST("/apps/info/:appname", requireSession, updateAppInfo)
  api.POST("/apps/show/:appname", requireSession, handleShowApp)
  api.POST("/apps/hide/:appname", requireSession, handleHideApp)
  api.POST("/apps/upload/:appname/:filetype", requireSession, handleAppUpload)
  api.DELETE("/apps/upload/:appname/:filetype", requireSession, handleAppDelete)
  api.POST("/apps/screenshot/:appname/:screenshot", requireSession, handleAppScreenshotUpload)
  api.DELETE("/apps/screenshot/:appname/:screenshot", requireSession, handleAppScreenshotDelete)
  api.GET("/apps/files/:appname", requireSession, handleAppFileList)
  api.GET("/apps/file/:appname", requireSession, handleAppFileGet)
  api.POST("/apps/file/:appname", requireSession, handleAppFileSave)
  api.DELETE("/apps/file/:appname", requireSession, handleAppFileDelete)
  api.POST("/apps/dir/:appname", requireSession, handleAppDirCreate)
  api.DELETE("/apps/dir/:appname", requireSession, handleAppDirDelete)
  var apps = r.Group("/apps")
  apps.Use(noCORS)
  apps.GET("/info/:appname", handleAppInfo)
  apps.GET("/icon/:appname", handleAppIcon)
  apps.GET("/download/:appname/:filetype", canAuth, handleAppFile)
  apps.GET("/screenshots/:appname/:screenshot", handleAppScreenshot)
  apps.GET("/all", handleAllApps)
  apps.GET("/featured", handleFeatured)
  apps.GET("/reviews/:appname", handleGetAppReviews)
  apps.POST("/review/:appname", handleAppReview)
  apps.GET("/owned", canAuth, handleOwnedApps)
  r.Run((":" + port))
}

