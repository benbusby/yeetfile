create table if not exists invites
(
    email     text  not null
        constraint invites_pk
            primary key,
    code_hash bytea not null
);
