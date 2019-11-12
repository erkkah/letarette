alter table docs rename column updated to "updated_deleted_" || strftime('%s.%f','now');
