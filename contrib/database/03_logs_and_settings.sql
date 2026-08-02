create table logs
(
    id bigint unsigned auto_increment
        primary key,
    cpe_uuid varchar(36) not null default '',
    full_xml longtext not null,
    code varchar(50) not null default '',
    message varchar(2000) not null default '',
    type varchar(30) not null default 'INFO',
    `from` varchar(10) not null default 'acs',
    session_id varchar(64) not null default '',
    detail longtext null,
    created_at datetime(6) default current_timestamp(6) not null
);

create index logs_cpe_uuid_created_at_index
    on logs (cpe_uuid, created_at);

create index logs_type_index
    on logs (type);

alter table cpe
    add column debug bool not null default false;
