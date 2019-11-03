package letarette

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

/*
with matches as (
select
	rowid,
	--replace(snippet(fts, 0, "(", ")", "...", 8), X'0A', " ") as snippet,
	rank as r
from
	fts
where
	fts match 'NEAR(london city limits)'
limit 5000
),
stats as (
select count(*) as cnt from matches
)
select spaces.space, docs.docID, matches.r, stats.cnt, replace(substr(docs.txt, 0, 25), X'0A', " ")||"..."
from matches
join docs on docs.id = matches.rowid
left join spaces using (spaceID)
cross join stats
where docs.alive
and space in ("wp")
order by r asc limit 10 offset 0;
*/

func (db *database) search(ctx context.Context, phrase string, spaces []string, limit uint16, offset uint16) ([]protocol.SearchResult, error) {
	const left = "\u3016"
	const right = "\u3017"
	const ellipsis = "\u2026"

	query := `
	select
		spaces.space as space,
		docs.docID as id,
		replace(snippet(fts, 0, :left, :right, :ellipsis, 8), X'0A', " ") as snippet,
		rank
	from 
		fts
		join docs on fts.rowid = docs.id
		left join spaces on docs.spaceID = spaces.spaceID
	where
		fts match '"%s"'
		and docs.alive
		and spaces.space in (?)
	order by rank asc limit :limit offset :offset
	`

	logger.Debug.Printf("Search query: [%s]", query)

	var result []protocol.SearchResult

	spaceArgs := make([]interface{}, len(spaces))
	for i, v := range spaces {
		spaceArgs[i] = v
	}
	spacedQuery, spacedArgs, err := sqlx.In(query, spaceArgs...)
	if err != nil {
		return result, fmt.Errorf("Failed to expand 'in' values: %w", err)
	}

	namedQuery, namedArgs, err := sqlx.Named(spacedQuery, map[string]interface{}{
		"left":     left,
		"right":    right,
		"ellipsis": ellipsis,
		"limit":    limit,
		"offset":   offset,
	})
	if err != nil {
		return result, fmt.Errorf("Failed to expand named binds: %w", err)
	}

	args := append(namedArgs[:0:0], namedArgs[:3]...)
	args = append(args, spacedArgs...)
	args = append(args, namedArgs[3:]...)

	phraseQuery := fmt.Sprintf(namedQuery, phrase)
	logger.Debug.Printf("Search query: [%s], args: %v", phraseQuery, args)
	err = db.rdb.SelectContext(ctx, &result, phraseQuery, args...)

	return result, err
}
