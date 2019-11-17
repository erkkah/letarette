with
matches as (
    select
        rowid,
        firstmatch(fts, 0) as matchColumn,
        firstmatch(fts, 1) as matchOffset,
        tokens(fts, firstmatch(fts, 0)) as numTokens,
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
    substr("…", 1, (matchOffset > 1)) ||
    replace(
        gettokens(fts,
            case matchColumn
                when 0 then docs.title
                when 1 then docs.txt
            end,
            max(matchOffset-1, 0), 10),
        X'0A', " "
    )
    || substr("…", 1, (numTokens > 10))
    as snippet
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
