alter table docs rename column updated to "deleted_" || strftime('%s.%f','now');
