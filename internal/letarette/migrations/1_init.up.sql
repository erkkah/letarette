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

-- Index meta data
create table if not exists meta (
    -- Unique id for this index
    indexID text not null,
    created timestamp default current_timestamp
);

-- Search spaces, each with their own index update position
create table if not exists spaces(
    spaceID integer primary key,
    space text not null unique,

    -- index position timestamp
    lastUpdatedAtNanos integer not null,
    -- index position documentID
    lastUpdatedDocID text not null default "",
    -- interest list creation timestamp
    listCreatedAtNanos integer not null default 0,

    check(
        lastUpdatedAtNanos <= listCreatedAtNanos
    )
);

-- The list of documents we are currently interested in getting updated
create table if not exists interest(
    spaceID integer not null,
    docID text not null,
    state integer not null,
    updatedNanos integer not null default 0,
    unique(spaceID, docID)
    foreign key (spaceID) references spaces(spaceID)
);

-- Document table with triggers to keep the full text index updated
create table if not exists docs(
    id integer primary key,
    spaceID integer not null,
    docID text not null,
    updatedNanos integer not null,
    title text not null,
    txt text not null,
    alive boolean not null default true,
    unique(spaceID, docID)
    foreign key (spaceID) references spaces(spaceID)
);

create index if not exists docs_spaceindex
on docs(spaceID);

create trigger if not exists docs_ai after insert on docs begin
    insert into fts(rowid, title, txt) values (new.id, new.title, new.txt);
end;

create trigger if not exists docs_ad after delete on docs begin
    insert into fts(fts, rowid, title, txt) values ('delete', old.id, old.title, old.txt);
end;

create trigger if not exists docs_au after update on docs begin
    insert into fts(fts, rowid, title, txt) values ('delete', old.id, old.title, old.txt);
    insert into fts(rowid, title, txt) values (new.id, new.title, new.txt);
end;

-- Current stemmer configuration
create table if not exists stemmerstate (
    languages text not null,
    removeDiacritics boolean not null,
    tokenCharacters text not null,
    separators text not null,
    updated timestamp default current_timestamp
);

create trigger if not exists stemmerstate_au after update
of languages, removeDiacritics, tokenCharacters, separators
on stemmerstate begin
    update stemmerstate set updated = current_timestamp;
end;

-- The full text index
create virtual table if not exists fts using fts5(
    title, txt, content='docs', content_rowid='id',
    tokenize='snowball', prefix='2 3 4'
);
