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

create table if not exists synonyms (
    id integer primary key,
    description text not null
);

create table if not exists synonym_words (
    synonymID integer,
    word text not null,
    unique (word),
    foreign key(synonymID) references synonyms(id) on delete cascade
);

insert into synonyms(description) values("one");
insert into synonym_words(synonymID, word) values (1, "1"), (1, "one");
