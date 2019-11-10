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
    space, r as rank, cnt as total, joined.docID as id,
    substr("…", 1, (first > 0))||replace(gettokens(fts, docs.txt, max(first-1, 0), 10), X'0A', " ")||"…" as snippet
from (
    select
        space, first, r, stats.cnt, docs.docID, docs.id
    from
        matches
        left join docs on docs.id = matches.rowid
        cross join stats
        join spaces using(spaceID)
    where
        space in (?)
        and docs.alive
    order by r asc
    limit :limit
    offset :offset
) joined
left join docs using(id)
left join fts on fts.rowid = (select id from docs limit 1)
