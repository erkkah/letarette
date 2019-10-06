alter table docs rename column alive to "deleted_" || strftime('%s.%f','now');
