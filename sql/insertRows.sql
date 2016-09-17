begin
	for i in 1 .. 99999
	loop 
	  insert into my_tbl (id,val,status)
	  select sequ_id.nextval,user||','||'Row #'||to_number(sequ_id.nextval) val, 'new'
	  from all_objects join (select 1 from all_objects where rownum<=100) r on 1=1
	  where rownum <=i;

	  commit;
	  dbms_lock.sleep(3);

	end loop; 
end;
/
