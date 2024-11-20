-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- This migration adds storage support for data sources. The only
-- constraints we enforce at the database layer are
--
-- * functions can only reference one data source, and must be deleted
--   if the data source is deleted
-- * data sources are tied to a project, and must be deleted if the
--   project is deleted
-- * rule types can reference one or more data source, and we want to
--   prevent deletion of a data source if there's a rule type
--   referencing it
--
-- The first two are simple foreign keys, while the third one is
-- enforced by the lack of `ON DELETE ...` clause in the
-- `rule_type_data_sources` table.
--
-- We also want to prevent the creation of a data source with a given
-- name if another data source with the same name exists in the
-- project hierarchy. I'm not sure how to express this as a database
-- constraint, nor I believe this would be efficient, so we decided to
-- let the application layer enforce that as we do with profiles.

CREATE TABLE data_sources(
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    project_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX data_sources_name_lower_idx ON data_sources (project_id, lower(name));

CREATE TABLE data_sources_functions(
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    data_source_id UUID NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (data_source_id) REFERENCES data_sources(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX data_sources_functions_name_lower_idx ON data_sources_functions (data_source_id, lower(name));

CREATE TABLE rule_type_data_sources(
    rule_type_id UUID NOT NULL,
    data_sources_id UUID NOT NULL,
    FOREIGN KEY (rule_type_id) REFERENCES rule_type(id),
    FOREIGN KEY (data_sources_id) REFERENCES data_sources(id),
    UNIQUE (rule_type_id, data_sources_id)
);

COMMIT;
