drop table fts;

create virtual table if not exists fts using fts5(
    txt, content='docs', content_rowid='id',
    tokenize='snowball'
);

insert into fts(fts) values('rebuild');
