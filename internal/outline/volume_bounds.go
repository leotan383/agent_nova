package outline

import (
	"os"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

type volumeBounds struct {
	From int
	To   int
}

func loadVolumeOutlineBounds(p *project.Project) (vols []int, mins, maxs map[int]int, err error) {
	vols, err = ListVolumeNumbers(p)
	if err != nil {
		return nil, nil, nil, err
	}
	mins = map[int]int{}
	maxs = map[int]int{}
	for _, v := range vols {
		data, _ := os.ReadFile(p.VolumeOutlinePath(v))
		for _, e := range ParseVolumeOutline(v, string(data)) {
			if mins[v] == 0 || e.Chapter < mins[v] {
				mins[v] = e.Chapter
			}
			if e.Chapter > maxs[v] {
				maxs[v] = e.Chapter
			}
		}
	}
	return vols, mins, maxs, nil
}

func chapterRangeForVolume(volume int, vols []int, mins, maxs map[int]int, maxBody int) volumeBounds {
	from := 1
	for _, v := range vols {
		if v >= volume {
			break
		}
		if maxs[v] > 0 && maxs[v]+1 > from {
			from = maxs[v] + 1
		}
	}
	if mins[volume] > 0 {
		from = mins[volume]
	}

	to := 0
	for _, v := range vols {
		if v > volume && mins[v] > 0 {
			to = mins[v] - 1
			break
		}
	}
	if to <= 0 {
		to = maxBody
		if maxs[volume] > to {
			to = maxs[volume]
		}
	}
	if maxs[volume] > 0 && maxs[volume] > to {
		to = maxs[volume]
	}
	if to < from {
		to = from
	}
	return volumeBounds{From: from, To: to}
}

func synthesizeRowsForEmptyOutline(
	volume int,
	vols []int,
	mins, maxs map[int]int,
	bodySet map[int]struct{},
	chMap map[int]store.Chapter,
) []Row {
	maxBody := 0
	for n := range bodySet {
		if n > maxBody {
			maxBody = n
		}
	}
	bounds := chapterRangeForVolume(volume, vols, mins, maxs, maxBody)
	if bounds.To < bounds.From {
		return nil
	}
	rows := make([]Row, 0, bounds.To-bounds.From+1)
	for chNum := bounds.From; chNum <= bounds.To; chNum++ {
		_, hasBody := bodySet[chNum]
		ch := chMap[chNum]
		match := MatchUnwritten
		if hasBody {
			match = MatchMatched
		}
		rows = append(rows, Row{
			Volume: volume, Chapter: chNum, Title: ch.Title,
			MatchStatus: match, HasBody: hasBody,
			WordCount: ch.WordCount, BodyStatus: ch.Status,
			InOutline: false,
		})
	}
	return rows
}
