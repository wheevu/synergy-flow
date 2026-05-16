-- +goose Up
alter table notifications add column if not exists resource_type text;
alter table notifications add column if not exists resource_id uuid;

-- +goose Down
alter table notifications drop column if exists resource_id;
alter table notifications drop column if exists resource_type;
