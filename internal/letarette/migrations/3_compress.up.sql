create view if not exists cdocs (
    id, title, txt
) as
select id, title, uncompress(txt) from docs;

drop table fts;

create virtual table fts using fts5(
    title, txt, content='cdocs', content_rowid='id',
    tokenize='snowball', prefix='2 3 4'
);

drop trigger docs_ai;

create trigger docs_ai after insert on docs begin
    insert into fts(rowid, title, txt) values (new.id, new.title, uncompress(new.txt));
end;

drop trigger docs_ad;

create trigger docs_ad after delete on docs begin
    insert into fts(fts, rowid, title, txt) values ('delete', old.id, old.title, uncompress(old.txt));
end;

drop trigger docs_au;

create trigger docs_au after update on docs begin
    insert into fts(fts, rowid, title, txt) values ('delete', old.id, old.title, uncompress(old.txt));
    insert into fts(rowid, title, txt) values (new.id, new.title, uncompress(new.txt));
end;

insert into fts(fts) values("rebuild");
