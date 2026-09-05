package insights

import "testing"

func TestCountsShortenAtEachThreshold(t *testing.T) {
	cases := []struct {
		given  int64
		wanted string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1k"},
		{1500, "1.5k"},
		{12500, "12.5k"},
		{1_000_000, "1M"},
		{2_400_000_000, "2.4B"},
		{-1500, "-1.5k"},
	}

	for _, held := range cases {
		if found := readableCount(held.given); found != held.wanted {
			t.Errorf("readableCount(%d) gave %q, wanted %q", held.given, found, held.wanted)
		}
	}
}

func TestChangeReadsAsNewOnlyWhenThereWasNothingBefore(t *testing.T) {
	cases := []struct {
		now    int64
		before int64
		change string
		kind   string
	}{
		{0, 0, "", ChangeSteady},
		{10, 0, NewChange, ChangeUp},
		{10, 10, SteadyChange, ChangeSteady},
		{20, 10, "+100.0%", ChangeUp},
		{5, 10, "-50.0%", ChangeDown},
	}

	for _, held := range cases {
		change, kind := readableChange(held.now, held.before)

		if change != held.change || kind != held.kind {
			t.Errorf(
				"readableChange(%d, %d) gave (%q, %q), wanted (%q, %q)",
				held.now, held.before, change, kind, held.change, held.kind,
			)
		}
	}
}

func TestToneFollowsWhetherMoreIsBetter(t *testing.T) {
	cases := []struct {
		kind   string
		better string
		wanted string
	}{
		{ChangeUp, MoreIsBetter, GoodTone},
		{ChangeDown, MoreIsBetter, BadTone},
		{ChangeUp, LessIsBetter, BadTone},
		{ChangeDown, LessIsBetter, GoodTone},
		{ChangeUp, NeitherIsBetter, PlainTone},
		{ChangeSteady, MoreIsBetter, PlainTone},
	}

	for _, held := range cases {
		if found := toneOf(held.kind, held.better); found != held.wanted {
			t.Errorf("toneOf(%s, %s) gave %s, wanted %s", held.kind, held.better, found, held.wanted)
		}
	}
}

func TestDurationsSwitchToSecondsAboveAThousand(t *testing.T) {
	cases := []struct {
		given  float64
		wanted string
	}{
		{0.07, "0.07 ms"},
		{12.5, "12.5 ms"},
		{999, "999 ms"},
		{1000, "1 sec"},
		{2500, "2.5 sec"},
	}

	for _, held := range cases {
		if found := readableDuration(held.given); found != held.wanted {
			t.Errorf("readableDuration(%v) gave %q, wanted %q", held.given, found, held.wanted)
		}
	}
}
