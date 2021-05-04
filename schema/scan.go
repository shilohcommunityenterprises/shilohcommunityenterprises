package schema

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/uptrace/bun/internal"
	"github.com/vmihailenco/msgpack/v5"
)

var scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

type ScannerFunc func(dest reflect.Value, src interface{}) error

var scanners = []ScannerFunc{
	reflect.Bool:          scanBool,
	reflect.Int:           scanInt64,
	reflect.Int8:          scanInt64,
	reflect.Int16:         scanInt64,
	reflect.Int32:         scanInt64,
	reflect.Int64:         scanInt64,
	reflect.Uint:          scanUint64,
	reflect.Uint8:         scanUint64,
	reflect.Uint16:        scanUint64,
	reflect.Uint32:        scanUint64,
	reflect.Uint64:        scanUint64,
	reflect.Uintptr:       scanUint64,
	reflect.Float32:       scanFloat64,
	reflect.Float64:       scanFloat64,
	reflect.Complex64:     nil,
	reflect.Complex128:    nil,
	reflect.Array:         nil,
	reflect.Chan:          nil,
	reflect.Func:          nil,
	reflect.Interface:     nil,
	reflect.Map:           scanJSON,
	reflect.Ptr:           nil,
	reflect.Slice:         scanJSON,
	reflect.String:        scanString,
	reflect.Struct:        nil,
	reflect.UnsafePointer: nil,
}

func FieldScanner(field *Field) ScannerFunc {
	if field.Tag.HasOption("msgpack") {
		return scanMsgpack
	}
	if field.Tag.HasOption("json_use_number") {
		return scanJSONUseNumber
	}
	return Scanner(field.Type)
}

func Scanner(typ reflect.Type) ScannerFunc {
	if typ.Implements(scannerType) {
		return scanScanner
	}

	kind := typ.Kind()

	if kind != reflect.Ptr {
		ptr := reflect.PtrTo(typ)
		if ptr.Implements(scannerType) {
			return addrScanner(scanScanner)
		}
	}

	switch typ {
	case timeType:
		return scanTime
	}

	return scanners[kind]
}

func scanBool(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case nil:
		dest.SetBool(false)
		return nil
	case bool:
		dest.SetBool(src)
		return nil
	case int64:
		dest.SetBool(src != 0)
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type(), dest)
}

func scanInt64(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case nil:
		dest.SetInt(0)
		return nil
	case int64:
		dest.SetInt(src)
		return nil
	case uint64:
		dest.SetInt(int64(src))
		return nil
	case []byte:
		n, err := strconv.ParseInt(internal.String(src), 10, 64)
		if err != nil {
			return err
		}
		dest.SetInt(n)
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}

func scanUint64(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case nil:
		dest.SetUint(0)
		return nil
	case uint64:
		dest.SetUint(src)
		return nil
	case int64:
		dest.SetUint(uint64(src))
		return nil
	case []byte:
		n, err := strconv.ParseUint(internal.String(src), 10, 64)
		if err != nil {
			return err
		}
		dest.SetUint(n)
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}

func scanFloat64(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case nil:
		dest.SetFloat(0)
		return nil
	case float64:
		dest.SetFloat(src)
		return nil
	case []byte:
		f, err := strconv.ParseFloat(internal.String(src), 64)
		if err != nil {
			return err
		}
		dest.SetFloat(f)
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}

func scanString(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case nil:
		dest.SetString("")
		return nil
	case string:
		dest.SetString(src)
		return nil
	case []byte:
		dest.SetString(string(src))
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}

func scanTime(dest reflect.Value, src interface{}) error {
	switch src := src.(type) {
	case nil:
		dest.Set(reflect.ValueOf(time.Time{}))
		return nil
	case time.Time:
		dest.Set(reflect.ValueOf(src))
		return nil
	case string:
		tm, err := internal.ParseTime(src)
		if err != nil {
			return err
		}
		dest.Set(reflect.ValueOf(tm))
		return nil
	}
	return fmt.Errorf("bun: can't scan %#v into %s", src, dest.Type())
}

func scanScanner(dest reflect.Value, src interface{}) error {
	return dest.Interface().(sql.Scanner).Scan(src)
}

func scanMsgpack(dest reflect.Value, src interface{}) error {
	b, err := toBytes(src)
	if err != nil {
		return err
	}

	dec := msgpack.GetDecoder()
	defer msgpack.PutDecoder(dec)

	dec.Reset(bytes.NewReader(b))
	return dec.DecodeValue(dest)
}

func scanJSON(dest reflect.Value, src interface{}) error {
	b, err := toBytes(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, dest.Addr().Interface())
}

func scanJSONUseNumber(dest reflect.Value, src interface{}) error {
	b, err := toBytes(src)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	return dec.Decode(dest.Addr().Interface())
}

func addrScanner(fn ScannerFunc) ScannerFunc {
	return func(dest reflect.Value, src interface{}) error {
		if !dest.CanAddr() {
			return fmt.Errorf("bun: Scan(nonaddressable %T)", dest.Interface())
		}
		return fn(dest.Addr(), src)
	}
}

func toBytes(src interface{}) ([]byte, error) {
	switch src := src.(type) {
	case string:
		return internal.Bytes(src), nil
	case []byte:
		return src, nil
	default:
		return nil, fmt.Errorf("bun: got %T, wanted []byte or string", src)
	}
}