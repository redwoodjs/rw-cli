package cmd

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestCreate(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "../testdata/create",
	})
}
