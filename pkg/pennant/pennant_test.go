package pennant_test

import (
	"testing"
	"time"

	"github.com/erkkah/letarette/pkg/pennant"
	"github.com/erkkah/letarette/pkg/xt"
)

func TestParseEmpty(t *testing.T) {
	xt := xt.X(t)

	empty := struct{}{}

	set, err := pennant.FromStruct(&empty)
	xt.NotNil(set)
	xt.Nil(err)

	err = set.Parse([]string{})
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
}

func TestParseUntaggedStringField(t *testing.T) {
	xt := xt.X(t)

	untagged := struct {
		StringField string
	}{}

	set, err := pennant.FromStruct(&untagged)
	xt.NotNil(set)
	xt.Nil(err)

	err = set.Parse([]string{"-stringField", "gremlin"})
	xt.Nil(err)

	xt.Equal(set.NFlag(), 1)
	xt.Equal(untagged.StringField, "gremlin")
}

func TestParseUntaggedIntegerField(t *testing.T) {
	xt := xt.X(t)

	untagged := struct {
		IntegerField int
	}{}

	set, err := pennant.FromStruct(&untagged)
	xt.NotNil(set)
	xt.Nil(err)

	err = set.Parse([]string{"-integerField", "4711"})
	xt.Nil(err)

	xt.Equal(set.NFlag(), 1)
	xt.Equal(untagged.IntegerField, 4711)
}

func TestParseDefaultIntegerField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		IntegerField int `default:"1234"`
	}{}

	set, err := pennant.FromStruct(&flags)
	xt.NotNil(set)
	xt.Nil(err)

	err = set.Parse([]string{})
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.Equal(flags.IntegerField, 1234)
}

func TestParseNamedIntegerField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		IntegerField int `default:"1234" name:"length"`
	}{}

	set, err := pennant.FromStruct(&flags)
	xt.NotNil(set)
	xt.Nil(err)

	err = set.Parse([]string{"-length", "999"})
	xt.Nil(err)

	xt.Equal(set.NFlag(), 1)
	xt.Equal(flags.IntegerField, 999)
}

func TestParseDefaultDurationField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		DurationField time.Duration `default:"1h" name:"duration"`
	}{}

	set, err := pennant.FromStruct(&flags)
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.Equal(flags.DurationField, time.Hour)
}

func TestParseDurationField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		DurationField time.Duration `default:"1h" name:"duration"`
	}{}

	set, err := pennant.FromStruct(&flags)
	xt.NotNil(set)
	xt.Nil(err)

	err = set.Parse([]string{"-duration", "32h"})
	xt.Nil(err)

	xt.Equal(set.NFlag(), 1)
	xt.Equal(flags.DurationField, time.Hour*32)
}

func TestParseArgField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		ArgField string `arg:"2"`
	}{}

	set, err := pennant.Parse(&flags, []string{"ett", "tu", "tre"})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.Equal(flags.ArgField, "tre")
}

func TestParseIntArgField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		ArgField int `arg:"0"`
	}{}

	set, err := pennant.Parse(&flags, []string{"90000"})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.Equal(flags.ArgField, 90000)
}

func TestParseDefaultIntArgField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		ArgField int `arg:"0" default:"889911"`
	}{}

	set, err := pennant.Parse(&flags, []string{})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.Equal(flags.ArgField, 889911)
}

func TestParseZeroIntArgField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		ArgField int `arg:"0"`
	}{}

	set, err := pennant.Parse(&flags, []string{})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.Equal(flags.ArgField, 0)
}

func TestParseArgsField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		ArgsField []string `args:"1"`
	}{}

	set, err := pennant.Parse(&flags, []string{"ett", "tu", "tre"})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.DeepEqual(flags.ArgsField, []string{"tu", "tre"})
}

func TestParseBoolField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		BoolField bool `name:"b"`
	}{}

	set, err := pennant.Parse(&flags, []string{"-b"})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 1)
	xt.DeepEqual(flags.BoolField, true)
}

func TestParseFalseBoolField(t *testing.T) {
	xt := xt.X(t)

	flags := struct {
		BoolField bool `name:"b"`
	}{}

	set, err := pennant.Parse(&flags, []string{})
	xt.NotNil(set)
	xt.Nil(err)

	xt.Equal(set.NFlag(), 0)
	xt.DeepEqual(flags.BoolField, false)
}
