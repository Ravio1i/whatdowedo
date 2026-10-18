package venues

import (
	"sort"
	"strings"
)

// Merge de-duplicates venues from several providers by normalised name,
// preferring entries that carry opening-hours information, and caps the list.
func Merge(limit int, lists ...[]Venue) []Venue {
	seen := map[string]int{}
	out := []Venue{}
	for _, l := range lists {
		for _, v := range l {
			key := normalise(v.Name)
			if key == "" {
				continue
			}
			if i, ok := seen[key]; ok {
				if out[i].OpeningHours == "" && v.OpeningHours != "" {
					out[i].OpeningHours, out[i].Status, out[i].HoursOnDate = v.OpeningHours, v.Status, v.HoursOnDate
				}
				if out[i].Rating == 0 && v.Rating != 0 {
					out[i].Rating = v.Rating
				}
				if out[i].Website == "" {
					out[i].Website = v.Website
				}
				continue
			}
			seen[key] = len(out)
			out = append(out, v)
		}
	}
	// Known-open first, then unknown, then closed; higher rating first.
	rank := map[OpenStatus]int{StatusOpen: 0, StatusUnknown: 1, StatusClosed: 2}
	sort.SliceStable(out, func(i, j int) bool {
		if rank[out[i].Status] != rank[out[j].Status] {
			return rank[out[i].Status] < rank[out[j].Status]
		}
		return out[i].Rating > out[j].Rating
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func normalise(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
