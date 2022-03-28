package pennant

import (
	"flag"
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type extendedFlagSet struct {
	flag.FlagSet
	namedArg []struct {
		ptr   reflect.Value
		index int
	}
	namedArgs []struct {
		ptr   reflect.Value
		start int
	}
}

// FromStruct creates a flag set from a tagged struct
func FromStruct(config any) (*flag.FlagSet, error) {
	flagSet, err := flagSetFromStruct(config)
	if err != nil {
		return nil, err
	}

	return &flagSet.FlagSet, nil
}

// Parse creates a flag set from a tagged struct and parses given arguments
func Parse(config any, args []string) (*flag.FlagSet, error) {
	set, err := flagSetFromStruct(config)
	if err != nil {
		return nil, err
	}

	err = set.Parse(args)
	if err != nil {
		return nil, err
	}

	for _, arg := range set.namedArg {
		argValue := set.Arg(arg.index)
		if len(argValue) == 0 {
			continue
		}
		err = assignFromString(arg.ptr, argValue)
		if err != nil {
			return nil, err
		}
	}

	for _, args := range set.namedArgs {
		argSlice := []string{}
		if set.NArg() >= args.start+1 {
			argSlice = set.Args()[args.start:]
		}
		args.ptr.Elem().Set(reflect.ValueOf(argSlice))
	}

	return &set.FlagSet, err
}

// MustParse calls Parse and exits with a helpful message on errors
func MustParse(config any, args []string) *flag.FlagSet {
	set, err := Parse(config, args)
	if err != nil {
		if set != nil {
			set.Usage()
		}
		fmt.Printf("failed to parse args: %v\n", err)
		os.Exit(1)
	}
	return set
}

func flagSetFromStruct(config any) (*extendedFlagSet, error) {
	configType := reflect.TypeOf(config)
	if configType == nil || configType.Kind() != reflect.Pointer || configType.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected config to be a pointer to struct")
	}

	flagSet := &extendedFlagSet{
		FlagSet: flag.FlagSet{
			Usage: flag.Usage,
		},
	}

	err := parseStructFields(reflect.ValueOf(config), "", flagSet)
	if err != nil {
		return nil, err
	}

	return flagSet, nil
}

func parseStructFields(structPointer reflect.Value, fieldPath string, set *extendedFlagSet) error {
	pointerType := structPointer.Type()
	if pointerType.Kind() != reflect.Pointer {
		return fmt.Errorf("expected pointer to struct")
	}

	structValue := structPointer.Elem()
	structType := pointerType.Elem()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		if field.Type.Kind() == reflect.Struct {
			err := parseStructFields(fieldValue.Addr(), path.Join(fieldPath, field.Name), set)
			if err != nil {
				return err
			}
		} else {
			err := defineFieldFlag(field, fieldValue.Addr(), fieldPath, set)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func assignFromString(fieldPointer reflect.Value, stringValue string) error {
	if fieldPointer.Elem().Type() == reflect.TypeOf(time.Second) {
		duration, err := time.ParseDuration(stringValue)
		if err != nil {
			return err
		}
		fieldPointer.Elem().Set(reflect.ValueOf(duration))
	} else {
		_, err := fmt.Sscanf(stringValue, "%v", fieldPointer.Interface())
		if err != nil {
			return err
		}
	}

	return nil
}

func defineFieldFlag(
	field reflect.StructField, fieldPointer reflect.Value, fieldPath string, set *extendedFlagSet,
) error {

	if fieldPointer.Type().Kind() != reflect.Pointer {
		return fmt.Errorf("expected pointer to field")
	}

	tag := field.Tag

	if defaultValue, ok := tag.Lookup("default"); ok {
		err := assignFromString(fieldPointer, defaultValue)
		if err != nil {
			return err
		}
	}

	if arg, ok := tag.Lookup("arg"); ok {
		idx, err := strconv.Atoi(arg)
		if err != nil {
			return err
		}
		set.namedArg = append(set.namedArg, struct {
			ptr   reflect.Value
			index int
		}{fieldPointer, idx})
		return nil
	}

	if args, ok := tag.Lookup("args"); ok {
		idx, err := strconv.Atoi(args)
		if err != nil {
			return err
		}
		set.namedArgs = append(set.namedArgs, struct {
			ptr   reflect.Value
			start int
		}{fieldPointer, idx})
		return nil
	}

	name := tag.Get("name")
	if name == "" {
		name = field.Name
	}

	flagName := buildFlagName(fieldPath, name)

	usage := tag.Get("usage")

	// Special handling for bool
	if field.Type.Kind() == reflect.Bool {
		set.BoolVar((*bool)(fieldPointer.UnsafePointer()), name, fieldPointer.Elem().Bool(), usage)
	} else {
		set.Func(flagName, usage, func(s string) error {
			return assignFromString(fieldPointer, s)
		})
	}

	return nil
}

func buildFlagName(path string, name string) string {
	parts := strings.Split(strings.ToLower(path), "/")
	name = string(strings.ToLower(name)[0]) + name[1:]
	parts = append(parts[1:], name)
	return strings.Join(parts, ".")
}
