create table if not exists stemmerstate (
    languages text not null,
    removeDiacritics boolean not null,
    tokenCharacters text not null,
    separators text not null,
    updated timestamp default current_timestamp
);

create trigger stemmerstate_au after update on stemmerstate begin
    update stemmerstate set updated = current_timestamp;
end;
