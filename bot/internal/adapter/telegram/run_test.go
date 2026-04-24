package telegram

import "testing"

func TestDisambigKeyboard_dedupesIdenticalButtonLabels(t *testing.T) {
	t.Parallel()
	k := disambigKeyboard([]string{
		"To Pimp A Butterfly | Kendrick Lamar (2015)",
		"To Pimp A Butterfly | Kendrick Lamar (2015)",
	})
	if k == nil {
		t.Fatal("nil keyboard")
	}
	albumRows := 0
	for _, row := range k.InlineKeyboard {
		if len(row) != 1 {
			continue
		}
		if row[0].Text == "Other" {
			continue
		}
		albumRows++
	}
	if albumRows != 1 {
		t.Fatalf("want 1 album button row, got %d (duplicate labels should not produce two same buttons)", albumRows)
	}
	var hasOther bool
	for _, row := range k.InlineKeyboard {
		if len(row) == 1 && row[0].Text == "Other" {
			hasOther = true
		}
	}
	if !hasOther {
		t.Fatal("expected Other button")
	}
}

func TestDisambigKeyboard_twoDistinctLabels_threeRows(t *testing.T) {
	t.Parallel()
	k := disambigKeyboard([]string{
		"Red | Taylor Swift (2012)",
		"Red | Gil Scott-Heron (1971)",
	})
	if k == nil {
		t.Fatal("nil keyboard")
	}
	if n := len(k.InlineKeyboard); n != 3 {
		t.Fatalf("want 3 rows (2 album + Other), got %d", n)
	}
}
