-- create table CreateHistory();
create table asset_request(
	id  varchar(255) not null,
	host_name varchar(255),
	resource_id varchar(255),
	ip_address varchar(255),
	asset_provider_id varchar(255),
	server_id varchar(255),
	received_on datetime,
	model_id varchar(255),
	status varchar(255),
	primary key(id)
)
engine=innodb; 

-- this table stores the module status configured.
create table module_status(
	id             varchar(255) not null,
	name           varchar(255),
	installed      boolean,
	configured     boolean,
    asset_request_id varchar(255) not null,
    primary key(id),
    KEY asset_request_id_fk (asset_request_id),
    CONSTRAINT asset_request_id_fk FOREIGN KEY (asset_request_id) REFERENCES asset_request (id)
)engine=innodb;
alter table module_status add unique index(name,asset_request_id);

