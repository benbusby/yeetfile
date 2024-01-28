
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_user WHERE usename = 'postgres') THEN
    CREATE USER postgres WITH PASSWORD '';
  END IF;
END $$;

GRANT ALL PRIVILEGES ON DATABASE postgres TO postgres;
set role to postgres;

create table if not exists metadata
(
    id       text not null
        constraint id
            primary key,
    chunks   integer,
    filename text,
    salt     bytea,
    b2_id    text,
    length   integer
);

create table if not exists b2_uploads
(
    metadata_id text not null
        constraint metadata_id
            primary key,
    upload_url  text,
    token       text,
    upload_id   text,
    checksums   text[]
);

create table if not exists expiry
(
    id        text not null
        constraint expiry_pk
            primary key,
    downloads integer,
    date      timestamp
);

create table if not exists users
(
    email      text,
    pw_hash    bytea,
    meter      bigint,
    id         text not null
        constraint users_pk
            primary key,
    payment_id text
);

create table if not exists stripe
(
    intent_id  text not null
        constraint stripe_pk
            primary key,
    account_id text,
    product_id text,
    quantity   integer,
    date       date
);

create table if not exists verify
(
    email   text not null
        constraint verification_pk
            primary key,
    code    text,
    date    date,
    pw_hash bytea
);
