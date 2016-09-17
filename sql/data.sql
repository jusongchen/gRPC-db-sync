create user demo identified by demo;
grant resource,connect to demo;

connect demo/demo
/

create sequence sequ_id;

drop table my_tbl;


create table my_tbl (
id number(9),
val char varying(2000),
status char varying (20)
);


insert into my_tbl (id,val,status)
select sequ_id.nextval,'Row #'||to_number(sequ_id.nextval)||','||user val, 'new'
from all_objects 
where rownum <=10;

commit;

select * from my_tbl;