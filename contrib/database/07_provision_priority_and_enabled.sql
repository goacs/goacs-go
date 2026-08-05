alter table provisions
    add column priority int not null default 0,
    add column enabled bool not null default true;

set @rownum = 0;
update provisions set priority = (@rownum := @rownum + 1) order by id;
