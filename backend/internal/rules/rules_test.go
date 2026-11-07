package rules

import (
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {
	entered, exited := Diff([]string{"b", "a"}, []string{"c", "b"})
	if !reflect.DeepEqual(entered, []string{"c"}) || !reflect.DeepEqual(exited, []string{"a"}) {
		t.Fatalf("entered=%v exited=%v", entered, exited)
	}
	entered, exited = Diff(nil, nil)
	if len(entered) != 0 || len(exited) != 0 {
		t.Fatalf("empty diff = %v %v", entered, exited)
	}
}

func TestMatches(t *testing.T) {
	if !Matches(Rule{EventType: "both"}, "veh_1", "exit") {
		t.Fatal("both should match exit")
	}
	vehicle := "veh_1"
	rule := Rule{VehicleID: &vehicle, EventType: "entry"}
	if !Matches(rule, "veh_1", "entry") {
		t.Fatal("expected entry match")
	}
	if Matches(rule, "veh_1", "exit") {
		t.Fatal("entry rule should ignore exit")
	}
	if Matches(rule, "veh_2", "entry") {
		t.Fatal("rule should ignore a different vehicle")
	}
}
