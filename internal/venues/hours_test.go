package venues

import (
	"testing"
	"time"
)

func TestEvaluateOSMHours(t *testing.T) {
	sat := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC) // Saturday
	sun := sat.AddDate(0, 0, 1)
	mon := sat.AddDate(0, 0, 2)
	cases := []struct {
		spec    string
		date    time.Time
		holiday bool
		want    OpenStatus
		hours   string
	}{
		{"24/7", sun, false, StatusOpen, "24/7"},
		{"Mo-Fr 09:00-18:00; Sa 10:00-14:00; Su off", sat, false, StatusOpen, "10:00-14:00"},
		{"Mo-Fr 09:00-18:00; Sa 10:00-14:00; Su off", sun, false, StatusClosed, ""},
		{"Mo-Fr 09:00-18:00; Sa 10:00-14:00", sun, false, StatusClosed, ""},
		{"Mo-Su 10:00-22:00", mon, false, StatusOpen, "10:00-22:00"},
		{"Mo,We,Fr 08:00-12:00,14:00-18:00", mon, false, StatusOpen, "08:00-12:00, 14:00-18:00"},
		{"10:00-20:00", sun, false, StatusOpen, "10:00-20:00"},
		{"Mo-Su 10:00-20:00; PH off", mon, true, StatusClosed, ""},
		{"Mo-Su 10:00-20:00; PH off", mon, false, StatusOpen, "10:00-20:00"},
		{"Sa-Mo 10:00-16:00", mon, false, StatusOpen, "10:00-16:00"},
		{"Apr-Oct Mo-Su 09:00-19:00", mon, false, StatusUnknown, ""},
		{"", mon, false, StatusUnknown, ""},
	}
	for _, c := range cases {
		got, hours := EvaluateOSMHours(c.spec, c.date, c.holiday)
		if got != c.want || hours != c.hours {
			t.Errorf("%q on %s: got (%s,%q) want (%s,%q)", c.spec, c.date.Weekday(), got, hours, c.want, c.hours)
		}
	}
}
