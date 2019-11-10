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
    substr("…", 1, (first > 0))||replace(gettokens(fts, docs.txt, max(first-1, 0), 10), X'0A', " ")||"…" as snippet
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
