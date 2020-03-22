-- Copyright 2020 Erik Agsjö
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
-- Title is used as matching snippet, further eliminating query-time work.

with
matches as (
    select
        rowid,
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
    space,
    r as rank,
    stats.cnt as total,
    docs.docID as id,
    docs.title as snippet
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

