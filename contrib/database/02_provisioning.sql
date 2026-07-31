create table provisions
(
    id int auto_increment
        primary key,
    name varchar(191) not null,
    events varchar(255) not null default '',
    requests varchar(255) not null default '',
    script longtext not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null,
    deleted_at datetime null
);

create table provision_rules
(
    id int auto_increment
        primary key,
    provision_id int not null,
    parameter varchar(255) not null,
    operator varchar(10) not null,
    value varchar(255) not null,
    created_at datetime default current_timestamp not null,
    updated_at datetime default current_timestamp not null,
    constraint provision_rules_provision_fk
        foreign key (provision_id) references provisions (id) on delete cascade
);

create index provision_rules_provision_id_index
    on provision_rules (provision_id);
