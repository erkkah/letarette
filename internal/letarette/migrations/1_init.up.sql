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

create table if not exists interest(
    spaceID integer not null,
    docID text not null,
    state integer not null,
    unique(spaceID, docID)
    foreign key (spaceID) references spaces(spaceID)
);

create table if not exists docs(
    id integer primary key,
    spaceID integer not null,
    docID text not null,
    updatedNanos integer not null,
    txt text not null,
    unique(spaceID, docID)
    foreign key (spaceID) references spaces(spaceID)
);

create virtual table if not exists fts using fts5(
    txt, content='docs', content_rowid='id',
    tokenize="porter unicode61 tokenchars '#'"
);

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
