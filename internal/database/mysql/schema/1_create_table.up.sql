create table pipeline
(
    id SERIAL PRIMARY KEY,
    name   VARCHAR(255) NOT NULL,
    spec TEXT NOT NULL DEFAULT('{}'),
    created_at BIGINT NOT NULL,
    updated_at BIGINT
);


CREATE UNIQUE INDEX uk_pipeline_name ON pipeline (name);


create table pipeline_run
(
    id SERIAL PRIMARY KEY,
    pipeline   TEXT NOT NULL,
    spec TEXT NOT NULL DEFAULT('{}'),
    status TEXT NOT NULL DEFAULT('{}'),
    state  VARCHAR(12) NOT NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT
);