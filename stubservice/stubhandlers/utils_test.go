package stubhandlers

import (
	"testing"
	"reflect"

	"github.com/mozilla-services/stubattribution/dmglib"
)

func TestModifyStubDMG(t *testing.T) {
	file, err := dmglib.OpenFile("../../testdata/attributable.dmg")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	defer file.Close()

	dmg, err := file.Parse()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	st := &stub{
		body: dmg.Data,
	}

	_, err = modifyStub(st, "hello=attribution", "osx")
	if err != nil {
		t.Errorf("modifyStub error: %s", err)
	}
}


func TestModifyStubFailOS(t *testing.T) {
	st := &stub{
		body: []byte("test"),
	}
	_, err := modifyStub(st, "hello=errors", "ardweeno")

	if err == nil {
		t.Errorf("expected an error, got: %s", reflect.TypeOf(err))
	}
}
