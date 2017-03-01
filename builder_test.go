package dba

import (
	"testing"
	"fmt"
)

func TestBuilder(t *testing.T) {
	var b = NewBuilder()
	b.Append("SELECT a.id, a.name")
	b.Append("FROM add AS a")
	b.Append("WHERE id>?", []interface{}{10}...)
	b.Append("LIMIT ?", 20)
	fmt.Println(b.ToSQL())
}