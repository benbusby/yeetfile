create table if not exists metadata
(
    id       text not null
        constraint id
            primary key,
    chunks   integer,
    filename text,
    b2_id    text,
    length   bigint
);

create table if not exists b2_uploads
(
    metadata_id text not null
        constraint metadata_id
            primary key,
    upload_url  text,
    token       text,
    upload_id   text,
    checksums   text[],
    local       boolean,
    name        text
);

create table if not exists expiry
(
    id        text not null
        constraint expiry_pk
            primary key,
    downloads smallint,
    date      timestamp
);

create table if not exists users
(
    id                  text not null
        constraint users_pk
            primary key,
    email               text,
    pw_hash             bytea,
    payment_id          text,
    last_upgraded_month smallint default 0,
    protected_key       bytea,
    public_key          bytea,
    storage_available   bigint   default 0,
    storage_used        bigint   default 0,
    send_available      bigint   default 0,
    send_used           bigint   default 0,
    upgrade_tag         text     default ''::text,
    upgrade_exp         timestamp,
    bandwidth           bigint   default 0,
    pw_hint             bytea    default '\x'::bytea,
    session_key         text     default ''::text,
    secret              bytea    default '\x'::bytea,
    recovery_hashes     text[]   default '{}'::text[]
);

create table if not exists stripe
(
    customer_id text not null
        constraint stripe_pk
            primary key,
    payment_id  text
        constraint stripe_uk
            unique,
    sub_id      text,
    created_at  timestamp
);

create table if not exists verify
(
    identity                   text not null
        constraint verification_pk
            primary key,
    code                       text,
    date                       timestamp,
    pw_hash                    bytea,
    protected_private_key      bytea,
    public_key                 bytea,
    protected_vault_folder_key bytea,
    pw_hint                    bytea,
    account_id                 text
);

create table if not exists vault
(
    id            text not null
        constraint vault_pk
            primary key,
    owner_id      text not null,
    name          text not null,
    b2_id         text    default ''::text,
    length        bigint,
    modified      timestamp,
    folder_id     text,
    chunks        integer,
    shared_by     text    default ''::text,
    protected_key bytea,
    link_tag      text    default ''::text,
    can_modify    boolean default true,
    ref_id        text,
    pw_data       bytea
);

create index if not exists vault_folder_id_index
    on vault (folder_id);

create table if not exists folders
(
    id            text  not null
        constraint folders_pk
            primary key,
    name          text  not null,
    owner_id      text  not null,
    protected_key bytea not null,
    shared_by     text    default ''::text,
    parent_id     text    default ''::text,
    modified      timestamp,
    link_tag      text    default ''::text,
    can_modify    boolean default true,
    ref_id        text    default ''::text,
    pw_folder     boolean default false
);

create index if not exists folders_id_index
    on folders (id);

create table if not exists sharing
(
    id           text not null
        constraint sharing_pk
            primary key,
    owner_id     text,
    recipient_id text,
    item_id      text,
    can_modify   boolean,
    is_folder    boolean
);

create table if not exists downloads
(
    id           text not null
        constraint downloads_pk
            primary key,
    file_id      text,
    user_id      text,
    chunk        integer default 0,
    total_chunks integer,
    updated      timestamp
);

create table if not exists forgot
(
    email     text not null
        constraint forgot_pk
            primary key,
    requested timestamp
);

create table if not exists change_email
(
    id         text not null
        constraint change_email_pk
            primary key,
    account_id text,
    old_email  text
        constraint change_email_pk2
            unique,
    date       timestamp
);

create table if not exists pass_index
(
    user_id   text not null
        constraint pass_index_pk
            primary key,
    enc_data  bytea,
    change_id integer
);

create table if not exists cron
(
    task_name    varchar(255) not null
        constraint cron_pk
            primary key,
    locked_until timestamp,
    last_run     timestamp
);
