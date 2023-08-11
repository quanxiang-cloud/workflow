create table `trigger`
(
    id SERIAL PRIMARY KEY,
    name  VARCHAR(255) NOT NULL,
    pipeline_name  VARCHAR(255) NOT NULL,
    type VARCHAR(24) NOT NULL,
    data TEXT NOT NULL ,
    created_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX uk_trigger_name ON `trigger` (name);

create table form_trigger
(
    id SERIAL PRIMARY KEY,
    name  VARCHAR(255) NOT NULL,
    pipeline_name  VARCHAR(255) NOT NULL,
    app_id VARCHAR(255) NOT NULL,
    table_id VARCHAR(255) NOT NULL,
    type VARCHAR(12) NOT NULL,
    created_at BIGINT NOT NULL
);

CREATE UNIQUE INDEX uk_trigger_form_trigger_name ON form_trigger (name);
CREATE UNIQUE INDEX uk_trigger_form_trigger ON form_trigger (app_id,table_id,pipeline_name);
CREATE INDEX uk_trigger_form_table_id ON form_trigger (table_id);

alter table form_trigger
    add filters text null;
