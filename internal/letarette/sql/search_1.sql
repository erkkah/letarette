-- Copyright 2019 Erik Agsjö
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Subquery - based strategy, trying to eliminate rows early.

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
                when 1 then uncompress(docs.txt)
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
-- re-joining on docs here is faster than pulling text and title into "joined" above
left join docs using(id)
-- Join in fts to get an fts handle to run "gettokens" on
left join fts on fts.rowid = (select id from docs limit 1)
