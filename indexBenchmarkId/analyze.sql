source /Users/svsuriya/VijaySuriya/prototypes/indexBenchmarkId/populate_votersId.sql
source /Users/svsuriya/VijaySuriya/prototypes/indexBenchmarkId/populate_votersUUID.sql

select database_name, table_name, index_name, stat_value*@@innodb_page_size from mysql.innodb_index_stats where stat_name='size' and database_name="vijay_suriya_db";

create index idx_age on voterUUID(age);
create index age_idx on voterId(age);