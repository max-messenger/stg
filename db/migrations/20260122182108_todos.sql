-- +migrate Up

CREATE TABLE IF NOT EXISTS todos
(
    id         serial primary key,
    user_id    int,
    message    varchar,
    done       bool,
    created_at timestamptz not null default current_timestamp
);

comment on table todos is 'Таблица для работы с todo листом';
comment on column todos.message is 'Содержание записи';

-- +migrate Down

DROP TABLE IF EXISTS todos;