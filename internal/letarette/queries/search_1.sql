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
    space, r as rank, cnt as total, joined.docID as id,
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
from (
    select
        space, matchColumn, matchOffset, numTokens, r, stats.cnt, docs.docID, docs.id
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
