package graph

// Muted and kept clear of the add/delete green and red so lanes never read as diff state.
var laneColors = []string{
	"#6aa5f0", "#e3a35c", "#b48be8", "#4fbfb5", "#e88aac",
	"#c9b75b", "#8fa0c0",
}

type Label struct {
	Name   string `json:"name"`
	Type   string `json:"type"` // "head", "branch", "remote", "tag"
	Remote string `json:"remote,omitempty"`
}

type GraphEdge struct {
	FromCol int    `json:"fromCol"`
	ToCol   int    `json:"toCol"`
	Color   string `json:"color"`
}

type CommitNode struct {
	Hash         string
	Subject      string
	Author       string
	Date         string
	Timestamp    int64
	ParentHashes []string
}

type GraphRow struct {
	Hash         string      `json:"hash"`
	ShortHash    string      `json:"shortHash"`
	Subject      string      `json:"subject"`
	Author       string      `json:"author"`
	Date         string      `json:"date"`
	Timestamp    int64       `json:"timestamp"`
	ParentHashes []string    `json:"parentHashes"`
	Labels       []Label     `json:"labels"`
	Column       int         `json:"column"`
	Color        string      `json:"color"`
	EdgesIn      []GraphEdge `json:"edgesIn"`
	Edges        []GraphEdge `json:"edges"`
	MaxColumn    int         `json:"maxColumn"`
}

func BuildGraph(commits []CommitNode, labelMap map[string][]Label) []GraphRow {
	lanes := make([]string, 0, 8)
	laneColorMap := make(map[int]string)
	colorIdx := 0

	findLane := func(hash string) int {
		for i, h := range lanes {
			if h == hash {
				return i
			}
		}
		return -1
	}

	addLane := func(hash string) int {
		// Reuse empty slot first
		for i, h := range lanes {
			if h == "" {
				lanes[i] = hash
				laneColorMap[i] = laneColors[colorIdx%len(laneColors)]
				colorIdx++
				return i
			}
		}
		col := len(lanes)
		lanes = append(lanes, hash)
		laneColorMap[col] = laneColors[colorIdx%len(laneColors)]
		colorIdx++
		return col
	}

	rows := make([]GraphRow, 0, len(commits))

	for _, c := range commits {
		col := findLane(c.Hash)
		if col == -1 {
			col = addLane(c.Hash)
		}
		color := laneColorMap[col]

		var edges []GraphEdge

		// Pass-through for all other active lanes
		for j, h := range lanes {
			if j != col && h != "" {
				edges = append(edges, GraphEdge{FromCol: j, ToCol: j, Color: laneColorMap[j]})
			}
		}

		if len(c.ParentHashes) == 0 {
			lanes[col] = ""
		} else {
			p0 := c.ParentHashes[0]
			p0col := findLane(p0)
			if p0col == -1 || p0col == col {
				lanes[col] = p0
				p0col = col
			} else {
				// Parent already has a lane — converge to it
				lanes[col] = ""
			}
			edges = append(edges, GraphEdge{FromCol: col, ToCol: p0col, Color: color})

			for _, ph := range c.ParentHashes[1:] {
				phCol := findLane(ph)
				if phCol == -1 {
					phCol = addLane(ph)
				}
				edges = append(edges, GraphEdge{FromCol: col, ToCol: phCol, Color: color})
			}
		}

		maxCol := col
		for _, e := range edges {
			if e.FromCol > maxCol {
				maxCol = e.FromCol
			}
			if e.ToCol > maxCol {
				maxCol = e.ToCol
			}
		}

		labels := labelMap[c.Hash]
		if labels == nil {
			labels = []Label{}
		}

		rows = append(rows, GraphRow{
			Hash:         c.Hash,
			ShortHash:    c.Hash[:7],
			Subject:      c.Subject,
			Author:       c.Author,
			Date:         c.Date,
			Timestamp:    c.Timestamp,
			ParentHashes: c.ParentHashes,
			Labels:       labels,
			Column:       col,
			Color:        color,
			Edges:        edges,
			MaxColumn:    maxCol,
		})
	}

	// Second pass: populate EdgesIn from previous row's outgoing edges
	for i := 1; i < len(rows); i++ {
		rows[i].EdgesIn = rows[i-1].Edges
	}

	return rows
}
