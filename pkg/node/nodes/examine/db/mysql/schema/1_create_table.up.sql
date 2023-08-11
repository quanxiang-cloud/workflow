create table examine_node
(
    id            varchar(200) PRIMARY KEY,
    task_id       varchar(200),
    flow_id       varchar(200),
    user_id       varchar(200),
    substitute    varchar(200),
    created_by    varchar(200),
    updated_by    varchar(200),
    examine_type  varchar(200),
    result        varchar(200),
    node_result   varchar(200),
    created_at    bigint,
    updated_at    bigint,
    app_id        varchar(200),
    form_table_id varchar(200),
    form_data_id  varchar(200),
    form_ref      varchar(200),
    urge_times    integer,
    remark        text,
    node_def_key  varchar(200)
);
