create table if not exists tasks(
  id integer primary key, profile text not null, text text not null,
  done integer not null default 0, pinned integer not null default 0,
  tag text, pos real not null, created real not null,
  parent_id integer, notes text, done_at real);
create table if not exists kv(k text primary key, v text not null);
