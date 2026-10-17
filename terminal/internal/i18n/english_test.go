package i18n

import (
	"reflect"
	"testing"
)

func TestEveryEnglishStringIsSet(t *testing.T) {
	set := reflect.ValueOf(*EnglishTranslationSet())
	for i := 0; i < set.NumField(); i++ {
		if set.Field(i).String() == "" {
			t.Errorf("%s has no English words", set.Type().Field(i).Name)
		}
	}
}
