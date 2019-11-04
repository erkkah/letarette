package letarette

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

func phrasesToMatchString(phrases []Phrase) string {
	var includes []string
	var excludes []string

	for _, v := range phrases {
		phraseExpr := v.Text
		if v.Wildcard {
			phraseExpr += "*"
		}
		if v.Exclude {
			excludes = append(excludes, phraseExpr)
		} else {
			includes = append(includes, phraseExpr)
		}
	}

	const nearRange = 15
	matchString := ""
	if len(includes) > 0 {
		matchString = fmt.Sprintf("NEAR(%s, %d)", strings.Join(includes, " "), nearRange)
	}
	if len(excludes) > 0 {
		matchString += fmt.Sprintf(" NOT (%s)", strings.Join(excludes, " OR "))
	}

	return matchString
}

func (db *database) search(ctx context.Context, phrases []Phrase, spaces []string, limit uint16, offset uint16) (protocol.SearchResult, error) {
	const left = "\u3016"
	const right = "\u3017"
	const ellipsis = "\u2026"

	if len(phrases) == 0 {
		return protocol.SearchResult{}, fmt.Errorf("Empty search phrase list")
	}

	matchString := phrasesToMatchString(phrases)

	query := `
	with
	matches as (
		select
			rowid,
			firstmatch(fts) as first,
			rank as r
		from
			fts
		where
			fts match :match
		limit :cap
	),
	stats as (
		select count(*) as cnt from matches
	)
	select
		spaces.space, docs.docID as id, matches.r as rank, stats.cnt as total,
		replace(gettokens(fts, docs.txt, first, 10), X'0A', " ")||:ellipsis as snippet
	from
		matches
		join docs on docs.id = matches.rowid
		left join fts on fts.rowid = (select id from docs limit 1)
		left join spaces using (spaceID)
		cross join stats
	where
		docs.alive
		and space in (?)
	order by matches.r asc limit :limit offset :offset;
	`

	type hit struct {
		protocol.SearchHit
		Total int
	}
	var hits []hit

	var result protocol.SearchResult

	spaceArgs := make([]interface{}, len(spaces))
	for i, v := range spaces {
		spaceArgs[i] = v
	}
	spacedQuery, spacedArgs, err := sqlx.In(query, spaceArgs...)
	if err != nil {
		return result, fmt.Errorf("Failed to expand 'in' values: %w", err)
	}

	namedQuery, namedArgs, err := sqlx.Named(spacedQuery, map[string]interface{}{
		"match":    matchString,
		"cap":      db.resultCap + 1,
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

	logger.Debug.Printf("Search query: [%s], args: %v", namedQuery, args)
	err = db.rdb.SelectContext(ctx, &hits, namedQuery, args...)
	if err != nil {
		return result, err
	}

	if len(hits) > 0 {
		result.TotalHits = hits[0].Total
	}
	if result.TotalHits > db.resultCap {
		result.TotalHits = db.resultCap
		result.Capped = true
	}
	result.Hits = make([]protocol.SearchHit, len(hits))
	for i, hit := range hits {
		result.Hits[i] = hit.SearchHit
	}

	return result, err
}
