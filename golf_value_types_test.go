package golf

import (
	"fmt"
	"testing"
)

func TestInt(t *testing.T) {
	f := Fore("hero", 3)
	assertMapLen(t, f, 1)
	assertKeyEquals(t, f, "hero", 3)
}

func TestString(t *testing.T) {
	f := Fore("hero", "three")
	assertMapLen(t, f, 1)
	assertKeyEquals(t, f, "hero", "three")
}

func TestStringThatGolfs(t *testing.T) {
	s := StringThatGolfs("three")
	f := Fore("hero", &s)
	t.Log(f)
	assertMapLen(t, f, 1)
	assertKeyEquals(t, f, "hero.golfer", "three three")
}

func TestNil(t *testing.T) {
	f := Fore("hero", nil)
	if f != nil {
		t.Fatal("not nil")
	}
}

type StringThatGolfs string

func (s *StringThatGolfs) GolfExportedFields() map[string]interface{} {
	return map[string]interface{}{"golfer": fmt.Sprintf("%s %s", *s, *s)}
}
