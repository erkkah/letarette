create table if not exists spaces(
    spaceID integer primary key,
    space text not null unique,

    -- timestamp of where we are in the index update process
    lastUpdate datetime not null,
    -- interest list creation timestamp
    listCreatedAt datetime,
    -- range of updated entries on interest list
    listUpdateStart datetime,
    listUpdateEnd datetime,
    chunkSize integer not null,
    -- offset into documents starting with the same timestamp
    chunkStart integer not null

    check(
        listUpdateStart <= listCreatedAt and
        listUpdateEnd <= listCreatedAt and
        lastUpdate <= listCreatedAt
    )
);

create table if not exists interest(
    spaceID integer not null,
    docID text not null,
    served integer not null,
    unique(spaceID, docID)
    foreign key (spaceID) references spaces(spaceID)
);

create virtual table if not exists docs using fts5(
    txt, spaceID unindexed, docID unindexed, updated unindexed,
    tokenize="porter unicode61 tokenchars '#'"
);
