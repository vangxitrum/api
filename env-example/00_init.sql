create user your_replication_username with replication encrypted password 'your_replication_password';

select pg_create_physical_replication_slot('your_replication_slot');
