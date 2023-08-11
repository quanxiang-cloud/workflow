create table old_flow
(
    id        varchar(200) PRIMARY KEY,
    flow_json text,
    status    varchar(200),
    app_id    varchar(200),
    form_id   varchar(200)
    created_by varchar (200)
) charset=utf8bm4;;


create table old_flow_var
(
    id             varchar(200) PRIMARY KEY,
    flow_id        varchar(200),
    code           varchar(200),
    default_value  varchar(200),
    `desc`         text,
    field_type     varchar(200),
    `name`         varchar(200),
    `type`         varchar(200),
    format         varchar(200),
    created_at     bigint,
    creator_avatar varchar(200),
    creator_id     varchar(200),
    updated_at     bigint,
    updator_name   varchar(200),
    updated_id     varchar(200)
) charset=utf8bm4;;
