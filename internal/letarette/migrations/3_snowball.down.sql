drop table fts;

create virtual table if not exists fts using fts5(
    txt, content='docs', content_rowid='id',
    tokenize="porter unicode61 tokenchars '#'"
);

insert into fts(fts) values('rebuild');
