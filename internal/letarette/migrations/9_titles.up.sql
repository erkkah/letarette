alter table docs add column title text;

drop table fts;

create virtual table if not exists fts using fts5(
    title, txt, content='docs', content_rowid='id',
    tokenize='snowball', prefix='2 3 4'
);

insert into fts(fts) values('rebuild');
insert into fts(fts, rank) values('rank', 'bm25(3.0, 1.0)');
