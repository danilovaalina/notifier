create type notification_channel as enum ('email', 'telegram');

create type notification_status as enum (
    'scheduled',
    'queued',
    'processing',
    'sent',
    'failed',
    'cancelled'
    );


create table if not exists notifications
(
    id             uuid primary key,
    channel        notification_channel not null,
    recipient      text                 not null,
    message        text                 not null,
    status         notification_status  not null,
    retry_count    bigint               not null default 0,
    scheduled_time timestamp            not null,
    created        timestamp            not null default now(),
    updated        timestamp            not null default now()
);