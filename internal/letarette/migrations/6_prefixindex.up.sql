drop table fts;

create virtual table if not exists fts using fts5(
    txt, content='docs', content_rowid='id',
    tokenize='snowball', prefix='2 3 4'
);

insert into fts(fts) values('rebuild');
