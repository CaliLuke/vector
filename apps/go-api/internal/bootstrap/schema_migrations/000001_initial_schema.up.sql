BEGIN;
--
-- PostgreSQL database dump
--

-- Dumped from database version 15.7
-- Dumped by pg_dump version 15.7

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA IF NOT EXISTS public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    provider_account_id character varying(255) NOT NULL,
    provider character varying NOT NULL,
    access_token text NOT NULL,
    access_token_expired_at timestamp with time zone,
    refresh_token text,
    refresh_token_expired_at timestamp with time zone,
    last_connected_at timestamp with time zone NOT NULL,
    metadata jsonb NOT NULL,
    user_id uuid NOT NULL,
    id_token text NOT NULL
);


--
-- Name: analytic_views; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.analytic_views (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    query jsonb NOT NULL,
    query_dict jsonb NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: api_activity_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_activity_logs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    token_identifier character varying(255) NOT NULL,
    path character varying(255) NOT NULL,
    method character varying(10) NOT NULL,
    query_params text,
    headers text,
    body text,
    response_code integer NOT NULL,
    response_body text,
    ip_address inet,
    user_agent character varying(512),
    created_by_id uuid,
    updated_by_id uuid,
    deleted_at timestamp with time zone,
    CONSTRAINT api_activity_logs_response_code_check CHECK ((response_code >= 0))
);


--
-- Name: api_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_tokens (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    token character varying(255) NOT NULL,
    label character varying(255) NOT NULL,
    user_type smallint NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid,
    description text NOT NULL,
    expired_at timestamp with time zone,
    is_active boolean NOT NULL,
    last_used timestamp with time zone,
    is_service boolean NOT NULL,
    deleted_at timestamp with time zone,
    allowed_rate_limit character varying(255) NOT NULL,
    CONSTRAINT api_tokens_user_type_check CHECK ((user_type >= 0))
);


--
-- Name: changelogs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.changelogs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    title character varying(255) NOT NULL,
    description text NOT NULL,
    version character varying(255) NOT NULL,
    tags jsonb NOT NULL,
    release_date timestamp with time zone,
    is_release_candidate boolean NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    deleted_at timestamp with time zone
);


--
-- Name: comment_reactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comment_reactions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    reaction text NOT NULL,
    actor_id uuid NOT NULL,
    comment_id uuid NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: cycle_issues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cycle_issues (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    cycle_id uuid NOT NULL,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: cycle_user_properties; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cycle_user_properties (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    filters jsonb NOT NULL,
    display_filters jsonb NOT NULL,
    display_properties jsonb NOT NULL,
    created_by_id uuid,
    cycle_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    rich_filters jsonb NOT NULL
);


--
-- Name: cycles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cycles (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    start_date timestamp with time zone,
    end_date timestamp with time zone,
    created_by_id uuid,
    owned_by_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    view_props jsonb NOT NULL,
    sort_order double precision NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    progress_snapshot jsonb NOT NULL,
    archived_at timestamp with time zone,
    logo_props jsonb NOT NULL,
    deleted_at timestamp with time zone,
    timezone character varying(255) NOT NULL,
    version integer NOT NULL
);


--
-- Name: deploy_boards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.deploy_boards (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    entity_identifier uuid,
    entity_name character varying(30),
    anchor character varying(255) NOT NULL,
    is_comments_enabled boolean NOT NULL,
    is_reactions_enabled boolean NOT NULL,
    is_votes_enabled boolean NOT NULL,
    view_props jsonb NOT NULL,
    created_by_id uuid,
    intake_id uuid,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    is_activity_enabled boolean NOT NULL,
    is_disabled boolean NOT NULL
);


--
-- Name: description_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.description_versions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    description_json jsonb NOT NULL,
    description_html text NOT NULL,
    description_binary bytea,
    description_stripped text,
    created_by_id uuid,
    description_id uuid NOT NULL,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: descriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.descriptions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    description_json jsonb NOT NULL,
    description_html text NOT NULL,
    description_binary bytea,
    description_stripped text,
    created_by_id uuid,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: device_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.device_sessions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    is_active boolean NOT NULL,
    user_agent character varying(255),
    ip_address inet,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone,
    created_by_id uuid,
    device_id uuid NOT NULL,
    session_id character varying(128) NOT NULL,
    updated_by_id uuid
);


--
-- Name: devices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.devices (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    device_id character varying(255),
    device_type character varying(255) NOT NULL,
    push_token character varying(255),
    is_active boolean NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL
);


--
-- Name: draft_issue_assignees; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_issue_assignees (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    assignee_id uuid NOT NULL,
    created_by_id uuid,
    draft_issue_id uuid NOT NULL,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: draft_issue_cycles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_issue_cycles (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    created_by_id uuid,
    cycle_id uuid NOT NULL,
    draft_issue_id uuid NOT NULL,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: draft_issue_labels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_issue_labels (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    created_by_id uuid,
    draft_issue_id uuid NOT NULL,
    label_id uuid NOT NULL,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: draft_issue_modules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_issue_modules (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    created_by_id uuid,
    draft_issue_id uuid NOT NULL,
    module_id uuid NOT NULL,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: draft_issues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_issues (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    name character varying(255),
    description_json jsonb NOT NULL,
    description_html text NOT NULL,
    description_stripped text,
    description_binary bytea,
    priority character varying(30) NOT NULL,
    start_date date,
    target_date date,
    sort_order double precision NOT NULL,
    completed_at timestamp with time zone,
    external_source character varying(255),
    external_id character varying(255),
    created_by_id uuid,
    estimate_point_id uuid,
    parent_id uuid,
    project_id uuid,
    state_id uuid,
    type_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: email_notification_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_notification_logs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    entity_identifier uuid,
    entity_name character varying(255) NOT NULL,
    data jsonb,
    processed_at timestamp with time zone,
    sent_at timestamp with time zone,
    entity character varying(200) NOT NULL,
    old_value character varying(300),
    new_value character varying(300),
    created_by_id uuid,
    receiver_id uuid NOT NULL,
    triggered_by_id uuid NOT NULL,
    updated_by_id uuid,
    deleted_at timestamp with time zone
);


--
-- Name: estimate_points; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.estimate_points (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    key integer NOT NULL,
    description text NOT NULL,
    value character varying(255) NOT NULL,
    created_by_id uuid,
    estimate_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: estimates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.estimates (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    type character varying(255) NOT NULL,
    last_used boolean NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: exporters; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exporters (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    project uuid[],
    provider character varying(50) NOT NULL,
    status character varying(50) NOT NULL,
    reason text NOT NULL,
    key text NOT NULL,
    url character varying(800),
    token character varying(255) NOT NULL,
    created_by_id uuid,
    initiated_by_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    filters jsonb,
    name character varying(255),
    type character varying(50) NOT NULL,
    deleted_at timestamp with time zone,
    rich_filters jsonb
);


--
-- Name: file_assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_assets (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    attributes jsonb NOT NULL,
    asset character varying(800) NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid,
    is_deleted boolean NOT NULL,
    deleted_at timestamp with time zone,
    is_archived boolean NOT NULL,
    comment_id uuid,
    entity_type character varying(255),
    external_id character varying(255),
    external_source character varying(255),
    is_uploaded boolean NOT NULL,
    issue_id uuid,
    page_id uuid,
    project_id uuid,
    size double precision NOT NULL,
    storage_metadata jsonb,
    user_id uuid,
    draft_issue_id uuid,
    entity_identifier character varying(255)
);


--
-- Name: github_comment_syncs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_comment_syncs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    repo_comment_id bigint NOT NULL,
    comment_id uuid NOT NULL,
    created_by_id uuid,
    issue_sync_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: github_issue_syncs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_issue_syncs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    repo_issue_id bigint NOT NULL,
    github_issue_id bigint NOT NULL,
    issue_url character varying(200) NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    repository_sync_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: github_repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_repositories (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(500) NOT NULL,
    url character varying(200),
    config jsonb NOT NULL,
    repository_id bigint NOT NULL,
    owner character varying(500) NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: github_repository_syncs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_repository_syncs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    credentials jsonb NOT NULL,
    actor_id uuid NOT NULL,
    created_by_id uuid,
    label_id uuid,
    project_id uuid NOT NULL,
    repository_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    workspace_integration_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: importers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.importers (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    service character varying(50) NOT NULL,
    status character varying(50) NOT NULL,
    metadata jsonb NOT NULL,
    config jsonb NOT NULL,
    data jsonb NOT NULL,
    created_by_id uuid,
    initiated_by_id uuid NOT NULL,
    project_id uuid NOT NULL,
    token_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    imported_data jsonb,
    deleted_at timestamp with time zone
);


--
-- Name: instance_admins; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance_admins (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    role integer NOT NULL,
    is_verified boolean NOT NULL,
    created_by_id uuid,
    instance_id uuid NOT NULL,
    updated_by_id uuid,
    user_id uuid,
    deleted_at timestamp with time zone,
    CONSTRAINT instance_admins_role_check CHECK ((role >= 0))
);


--
-- Name: instance_configurations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instance_configurations (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    key character varying(100) NOT NULL,
    value text,
    category text NOT NULL,
    is_encrypted boolean NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    deleted_at timestamp with time zone
);


--
-- Name: instances; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.instances (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    instance_name character varying(255) NOT NULL,
    whitelist_emails text,
    instance_id character varying(255) NOT NULL,
    current_version character varying(255) NOT NULL,
    last_checked_at timestamp with time zone NOT NULL,
    namespace character varying(255),
    is_telemetry_enabled boolean NOT NULL,
    is_support_required boolean NOT NULL,
    is_setup_done boolean NOT NULL,
    is_signup_screen_visited boolean NOT NULL,
    is_verified boolean NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    domain text NOT NULL,
    latest_version character varying(255),
    edition character varying(255) NOT NULL,
    deleted_at timestamp with time zone,
    is_test boolean NOT NULL,
    is_current_version_deprecated boolean NOT NULL
);


--
-- Name: intake_issues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.intake_issues (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    status integer NOT NULL,
    snoozed_till timestamp with time zone,
    source character varying(255),
    created_by_id uuid,
    duplicate_to_id uuid,
    intake_id uuid NOT NULL,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    deleted_at timestamp with time zone,
    extra jsonb NOT NULL,
    source_email text
);


--
-- Name: intakes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.intakes (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    is_default boolean NOT NULL,
    view_props jsonb NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    logo_props jsonb NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: integrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.integrations (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    title character varying(400) NOT NULL,
    provider character varying(400) NOT NULL,
    network integer NOT NULL,
    description jsonb NOT NULL,
    author character varying(400) NOT NULL,
    webhook_url text NOT NULL,
    webhook_secret text NOT NULL,
    redirect_url text NOT NULL,
    metadata jsonb NOT NULL,
    verified boolean NOT NULL,
    avatar_url text,
    created_by_id uuid,
    updated_by_id uuid,
    deleted_at timestamp with time zone,
    CONSTRAINT integrations_network_check CHECK ((network >= 0))
);


--
-- Name: issue_activities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_activities (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    verb character varying(255) NOT NULL,
    field character varying(255),
    old_value text,
    new_value text,
    comment text NOT NULL,
    attachments character varying(200)[] NOT NULL,
    created_by_id uuid,
    issue_id uuid,
    issue_comment_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    actor_id uuid,
    new_identifier uuid,
    old_identifier uuid,
    epoch double precision,
    deleted_at timestamp with time zone
);


--
-- Name: issue_assignees; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_assignees (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    assignee_id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_attachments (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    attributes jsonb NOT NULL,
    asset character varying(100) NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    deleted_at timestamp with time zone
);


--
-- Name: issue_blockers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_blockers (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    block_id uuid NOT NULL,
    blocked_by_id uuid NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_comments (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    comment_stripped text NOT NULL,
    attachments character varying(200)[] NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    actor_id uuid,
    comment_html text NOT NULL,
    comment_json jsonb NOT NULL,
    access character varying(100) NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    deleted_at timestamp with time zone,
    edited_at timestamp with time zone,
    description_id uuid,
    parent_id uuid
);


--
-- Name: issue_description_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_description_versions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    description_binary bytea,
    description_html text NOT NULL,
    description_stripped text,
    description_json jsonb NOT NULL,
    last_saved_at timestamp with time zone NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    owned_by_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: issue_labels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_labels (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    label_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_links (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    title character varying(255),
    url text NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    metadata jsonb NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_mentions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_mentions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    mention_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_reactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_reactions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    reaction text NOT NULL,
    actor_id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_relations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_relations (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    relation_type character varying(20) NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    related_issue_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_sequences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_sequences (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    sequence bigint NOT NULL,
    deleted boolean NOT NULL,
    created_by_id uuid,
    issue_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT issue_sequence_sequence_check CHECK ((sequence >= 0))
);


--
-- Name: issue_subscribers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_subscribers (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    subscriber_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issue_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_types (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    logo_props jsonb NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    is_active boolean NOT NULL,
    deleted_at timestamp with time zone,
    is_default boolean NOT NULL,
    level double precision NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    is_epic boolean NOT NULL
);


--
-- Name: issue_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_versions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    parent uuid,
    state uuid,
    estimate_point uuid,
    name character varying(255) NOT NULL,
    priority character varying(30) NOT NULL,
    start_date date,
    target_date date,
    sequence_id integer NOT NULL,
    sort_order double precision NOT NULL,
    completed_at timestamp with time zone,
    archived_at date,
    is_draft boolean NOT NULL,
    external_source character varying(255),
    external_id character varying(255),
    type uuid,
    last_saved_at timestamp with time zone NOT NULL,
    owned_by_id uuid NOT NULL,
    assignees uuid[] NOT NULL,
    labels uuid[] NOT NULL,
    cycle uuid,
    modules uuid[] NOT NULL,
    properties jsonb NOT NULL,
    meta jsonb NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    activity_id uuid
);


--
-- Name: issue_views; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_views (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    query jsonb NOT NULL,
    access smallint NOT NULL,
    filters jsonb NOT NULL,
    created_by_id uuid,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    display_filters jsonb NOT NULL,
    display_properties jsonb NOT NULL,
    sort_order double precision NOT NULL,
    logo_props jsonb NOT NULL,
    is_locked boolean NOT NULL,
    owned_by_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    rich_filters jsonb NOT NULL,
    archived_at timestamp with time zone,
    CONSTRAINT issue_views_access_check CHECK ((access >= 0))
);


--
-- Name: issue_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_votes (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    vote integer NOT NULL,
    actor_id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: issues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issues (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description_json jsonb NOT NULL,
    priority character varying(30) NOT NULL,
    start_date date,
    target_date date,
    sequence_id integer NOT NULL,
    created_by_id uuid,
    parent_id uuid,
    project_id uuid NOT NULL,
    state_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    description_html text NOT NULL,
    description_stripped text,
    completed_at timestamp with time zone,
    sort_order double precision NOT NULL,
    point integer,
    archived_at date,
    is_draft boolean NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    description_binary bytea,
    estimate_point_id uuid,
    type_id uuid,
    deleted_at timestamp with time zone
);


--
-- Name: labels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.labels (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    created_by_id uuid,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    parent_id uuid,
    color character varying(255) NOT NULL,
    sort_order double precision NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    deleted_at timestamp with time zone
);


--
-- Name: module_issues; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.module_issues (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    issue_id uuid NOT NULL,
    module_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: module_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.module_links (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    title character varying(255),
    url character varying(200) NOT NULL,
    created_by_id uuid,
    module_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    metadata jsonb NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: module_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.module_members (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    member_id uuid NOT NULL,
    module_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: module_user_properties; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.module_user_properties (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    filters jsonb NOT NULL,
    display_filters jsonb NOT NULL,
    display_properties jsonb NOT NULL,
    created_by_id uuid,
    module_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    rich_filters jsonb NOT NULL
);


--
-- Name: modules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.modules (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    description_text jsonb,
    description_html jsonb,
    start_date date,
    target_date date,
    status character varying(20) NOT NULL,
    created_by_id uuid,
    lead_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    view_props jsonb NOT NULL,
    sort_order double precision NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    archived_at timestamp with time zone,
    logo_props jsonb NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    data jsonb,
    entity_identifier uuid,
    entity_name character varying(255) NOT NULL,
    title text NOT NULL,
    message jsonb,
    message_html text NOT NULL,
    message_stripped text,
    sender character varying(255) NOT NULL,
    read_at timestamp with time zone,
    snoozed_till timestamp with time zone,
    archived_at timestamp with time zone,
    created_by_id uuid,
    project_id uuid,
    receiver_id uuid NOT NULL,
    triggered_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: page_labels; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_labels (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    label_id uuid NOT NULL,
    page_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: page_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_logs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    transaction uuid NOT NULL,
    entity_identifier uuid,
    entity_name character varying(30) NOT NULL,
    created_by_id uuid,
    page_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    entity_type character varying(30)
);


--
-- Name: page_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_versions (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    last_saved_at timestamp with time zone NOT NULL,
    description_binary bytea,
    description_html text NOT NULL,
    description_stripped text,
    description_json jsonb NOT NULL,
    created_by_id uuid,
    owned_by_id uuid NOT NULL,
    page_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    sub_pages_data jsonb NOT NULL
);


--
-- Name: pages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pages (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name text NOT NULL,
    description_json jsonb NOT NULL,
    description_html text NOT NULL,
    description_stripped text,
    access smallint NOT NULL,
    created_by_id uuid,
    owned_by_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    color character varying(255) NOT NULL,
    archived_at date,
    is_locked boolean NOT NULL,
    parent_id uuid,
    view_props jsonb NOT NULL,
    logo_props jsonb NOT NULL,
    description_binary bytea,
    is_global boolean NOT NULL,
    deleted_at timestamp with time zone,
    moved_to_page uuid,
    moved_to_project uuid,
    external_id character varying(255),
    external_source character varying(255),
    sort_order double precision NOT NULL,
    CONSTRAINT pages_access_check CHECK ((access >= 0))
);


--
-- Name: profiles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.profiles (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    theme jsonb NOT NULL,
    is_tour_completed boolean NOT NULL,
    onboarding_step jsonb NOT NULL,
    use_case text,
    role character varying(300),
    is_onboarded boolean NOT NULL,
    last_workspace_id uuid,
    billing_address_country character varying(255) NOT NULL,
    billing_address jsonb,
    has_billing_address boolean NOT NULL,
    company_name character varying(255) NOT NULL,
    user_id uuid NOT NULL,
    is_mobile_onboarded boolean NOT NULL,
    mobile_onboarding_step jsonb NOT NULL,
    mobile_timezone_auto_set boolean NOT NULL,
    language character varying(255) NOT NULL,
    is_smooth_cursor_enabled boolean NOT NULL,
    start_of_the_week smallint NOT NULL,
    is_app_rail_docked boolean NOT NULL,
    background_color character varying(255) NOT NULL,
    goals jsonb NOT NULL,
    has_marketing_email_consent boolean NOT NULL,
    is_navigation_tour_completed boolean NOT NULL,
    is_subscribed_to_changelog boolean NOT NULL,
    notification_view_mode character varying(255) NOT NULL,
    product_tour jsonb NOT NULL,
    CONSTRAINT profiles_start_of_the_week_check CHECK ((start_of_the_week >= 0))
);


--
-- Name: project_deploy_boards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_deploy_boards (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    anchor character varying(255) NOT NULL,
    comments boolean NOT NULL,
    reactions boolean NOT NULL,
    votes boolean NOT NULL,
    views jsonb NOT NULL,
    created_by_id uuid,
    intake_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: project_identifiers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_identifiers (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    name character varying(12) NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid,
    deleted_at timestamp with time zone
);


--
-- Name: project_identifier_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.project_identifiers ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.project_identifier_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: project_issue_types; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_issue_types (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    level integer NOT NULL,
    is_default boolean NOT NULL,
    created_by_id uuid,
    issue_type_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    CONSTRAINT project_issue_types_level_check CHECK ((level >= 0))
);


--
-- Name: project_member_invites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_member_invites (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    email character varying(255) NOT NULL,
    accepted boolean NOT NULL,
    token character varying(255) NOT NULL,
    message text,
    responded_at timestamp with time zone,
    role smallint NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT project_member_invite_role_check CHECK ((role >= 0))
);


--
-- Name: project_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_members (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    comment text,
    role smallint NOT NULL,
    created_by_id uuid,
    member_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    view_props jsonb NOT NULL,
    default_props jsonb NOT NULL,
    sort_order double precision NOT NULL,
    preferences jsonb NOT NULL,
    is_active boolean NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT project_member_role_check CHECK ((role >= 0))
);


--
-- Name: project_pages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_pages (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    page_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: project_public_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_public_members (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    created_by_id uuid,
    member_id uuid NOT NULL,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: project_user_properties; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_user_properties (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    display_properties jsonb NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    display_filters jsonb NOT NULL,
    filters jsonb NOT NULL,
    deleted_at timestamp with time zone,
    rich_filters jsonb NOT NULL,
    preferences jsonb NOT NULL,
    sort_order double precision NOT NULL
);


--
-- Name: project_webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_webhooks (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    webhook_id uuid NOT NULL,
    workspace_id uuid NOT NULL
);


--
-- Name: projects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.projects (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    description_text jsonb,
    description_html jsonb,
    network smallint NOT NULL,
    identifier character varying(12) NOT NULL,
    created_by_id uuid,
    default_assignee_id uuid,
    project_lead_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    emoji character varying(255),
    cycle_view boolean NOT NULL,
    module_view boolean NOT NULL,
    cover_image text,
    issue_views_view boolean NOT NULL,
    page_view boolean NOT NULL,
    estimate_id uuid,
    icon_prop jsonb,
    intake_view boolean NOT NULL,
    archive_in integer NOT NULL,
    close_in integer NOT NULL,
    default_state_id uuid,
    logo_props jsonb NOT NULL,
    archived_at timestamp with time zone,
    is_time_tracking_enabled boolean NOT NULL,
    is_issue_type_enabled boolean NOT NULL,
    deleted_at timestamp with time zone,
    guest_view_all_features boolean NOT NULL,
    timezone character varying(255) NOT NULL,
    cover_image_asset_id uuid,
    external_id character varying(255),
    external_source character varying(255),
    CONSTRAINT project_network_check CHECK ((network >= 0))
);


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    session_data text NOT NULL,
    expire_date timestamp with time zone NOT NULL,
    device_info jsonb,
    session_key character varying(128) NOT NULL,
    user_id character varying(50)
);


--
-- Name: slack_project_syncs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.slack_project_syncs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    access_token character varying(300) NOT NULL,
    scopes text NOT NULL,
    bot_user_id character varying(50) NOT NULL,
    webhook_url character varying(1000) NOT NULL,
    data jsonb NOT NULL,
    team_id character varying(30) NOT NULL,
    team_name character varying(300) NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    workspace_integration_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: social_login_connections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.social_login_connections (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    medium character varying(20) NOT NULL,
    last_login_at timestamp with time zone,
    last_received_at timestamp with time zone,
    token_data jsonb,
    extra_data jsonb,
    created_by_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: states; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.states (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    color character varying(255) NOT NULL,
    slug character varying(100) NOT NULL,
    created_by_id uuid,
    project_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    sequence double precision NOT NULL,
    "group" character varying(20) NOT NULL,
    "default" boolean NOT NULL,
    external_id character varying(255),
    external_source character varying(255),
    is_triage boolean NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: stickies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stickies (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    name text,
    description jsonb NOT NULL,
    description_html text NOT NULL,
    description_stripped text,
    description_binary bytea,
    logo_props jsonb NOT NULL,
    color character varying(255),
    background_color character varying(255),
    created_by_id uuid,
    owner_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    sort_order double precision NOT NULL
);


--
-- Name: teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teams (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    logo_props jsonb NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: user_favorites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_favorites (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    entity_type character varying(100) NOT NULL,
    entity_identifier uuid,
    name character varying(255),
    is_folder boolean NOT NULL,
    sequence double precision NOT NULL,
    created_by_id uuid,
    parent_id uuid,
    project_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: user_notification_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_notification_preferences (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    property_change boolean NOT NULL,
    state_change boolean NOT NULL,
    comment boolean NOT NULL,
    mention boolean NOT NULL,
    issue_completed boolean NOT NULL,
    created_by_id uuid,
    project_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid,
    deleted_at timestamp with time zone
);


--
-- Name: user_recent_visits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_recent_visits (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    entity_identifier uuid,
    entity_name character varying(30) NOT NULL,
    visited_at timestamp with time zone NOT NULL,
    created_by_id uuid,
    project_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    password character varying(128) NOT NULL,
    last_login timestamp with time zone,
    id uuid NOT NULL,
    username character varying(128) NOT NULL,
    mobile_number character varying(255),
    email character varying(255),
    first_name character varying(255) NOT NULL,
    last_name character varying(255) NOT NULL,
    avatar text NOT NULL,
    date_joined timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    last_location character varying(255) NOT NULL,
    created_location character varying(255) NOT NULL,
    is_superuser boolean NOT NULL,
    is_managed boolean NOT NULL,
    is_password_expired boolean NOT NULL,
    is_active boolean NOT NULL,
    is_staff boolean NOT NULL,
    is_email_verified boolean NOT NULL,
    is_password_autoset boolean NOT NULL,
    token character varying(64) NOT NULL,
    user_timezone character varying(255) NOT NULL,
    last_active timestamp with time zone,
    last_login_time timestamp with time zone,
    last_logout_time timestamp with time zone,
    last_login_ip character varying(255) NOT NULL,
    last_logout_ip character varying(255) NOT NULL,
    last_login_medium character varying(20) NOT NULL,
    last_login_uagent text NOT NULL,
    token_updated_at timestamp with time zone,
    is_bot boolean NOT NULL,
    cover_image character varying(800),
    display_name character varying(255) NOT NULL,
    avatar_asset_id uuid,
    cover_image_asset_id uuid,
    bot_type character varying(30),
    is_email_valid boolean NOT NULL,
    masked_at timestamp with time zone,
    is_password_reset_required boolean NOT NULL
);


--
-- Name: webhook_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhook_logs (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    event_type character varying(255),
    request_method character varying(10),
    request_headers text,
    request_body text,
    response_status text,
    response_headers text,
    response_body text,
    retry_count smallint NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    webhook uuid NOT NULL,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT webhook_logs_retry_count_check CHECK ((retry_count >= 0))
);


--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    url character varying(1024) NOT NULL,
    is_active boolean NOT NULL,
    secret_key character varying(255) NOT NULL,
    project boolean NOT NULL,
    issue boolean NOT NULL,
    module boolean NOT NULL,
    cycle boolean NOT NULL,
    issue_comment boolean NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    is_internal boolean NOT NULL,
    version character varying(50) NOT NULL
);


--
-- Name: workspace_home_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_home_preferences (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    key character varying(255) NOT NULL,
    is_enabled boolean NOT NULL,
    config jsonb NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    sort_order double precision NOT NULL
);


--
-- Name: workspace_integrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_integrations (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    metadata jsonb NOT NULL,
    config jsonb NOT NULL,
    actor_id uuid NOT NULL,
    api_token_id uuid NOT NULL,
    created_by_id uuid,
    integration_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: workspace_member_invites; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_member_invites (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    email character varying(255) NOT NULL,
    accepted boolean NOT NULL,
    token character varying(255) NOT NULL,
    message text,
    responded_at timestamp with time zone,
    role smallint NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    CONSTRAINT workspace_member_invite_role_check CHECK ((role >= 0))
);


--
-- Name: workspace_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_members (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    role smallint NOT NULL,
    created_by_id uuid,
    member_id uuid NOT NULL,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    company_role text,
    view_props jsonb NOT NULL,
    default_props jsonb NOT NULL,
    issue_props jsonb NOT NULL,
    is_active boolean NOT NULL,
    deleted_at timestamp with time zone,
    explored_features jsonb NOT NULL,
    getting_started_checklist jsonb NOT NULL,
    tips jsonb NOT NULL,
    CONSTRAINT workspace_member_role_check CHECK ((role >= 0))
);


--
-- Name: workspace_themes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_themes (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(300) NOT NULL,
    colors jsonb NOT NULL,
    actor_id uuid NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone
);


--
-- Name: workspace_user_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_user_links (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    title character varying(255),
    url text NOT NULL,
    metadata jsonb NOT NULL,
    created_by_id uuid,
    owner_id uuid NOT NULL,
    project_id uuid,
    updated_by_id uuid,
    workspace_id uuid NOT NULL
);


--
-- Name: workspace_user_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_user_preferences (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    deleted_at timestamp with time zone,
    id uuid NOT NULL,
    key character varying(255) NOT NULL,
    is_pinned boolean NOT NULL,
    sort_order double precision NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL
);


--
-- Name: workspace_user_properties; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace_user_properties (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    filters jsonb NOT NULL,
    display_filters jsonb NOT NULL,
    display_properties jsonb NOT NULL,
    created_by_id uuid,
    updated_by_id uuid,
    user_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    deleted_at timestamp with time zone,
    rich_filters jsonb NOT NULL,
    navigation_control_preference character varying(25) NOT NULL,
    navigation_project_limit integer NOT NULL
);


--
-- Name: workspaces; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspaces (
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    id uuid NOT NULL,
    name character varying(80) NOT NULL,
    logo text,
    slug character varying(48) NOT NULL,
    created_by_id uuid,
    owner_id uuid NOT NULL,
    updated_by_id uuid,
    organization_size character varying(20),
    deleted_at timestamp with time zone,
    logo_asset_id uuid,
    timezone character varying(255) NOT NULL,
    background_color character varying(255) NOT NULL
);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: accounts accounts_provider_provider_account_id_daac1f10_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_provider_provider_account_id_daac1f10_uniq UNIQUE (provider, provider_account_id);


--
-- Name: analytic_views analytic_views_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytic_views
    ADD CONSTRAINT analytic_views_pkey PRIMARY KEY (id);


--
-- Name: api_activity_logs api_activity_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_activity_logs
    ADD CONSTRAINT api_activity_logs_pkey PRIMARY KEY (id);


--
-- Name: api_tokens api_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_pkey PRIMARY KEY (id);


--
-- Name: api_tokens api_tokens_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_token_key UNIQUE (token);


--
-- Name: changelogs changelogs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelogs
    ADD CONSTRAINT changelogs_pkey PRIMARY KEY (id);


--
-- Name: comment_reactions comment_reactions_comment_id_actor_id_reac_24dc2de6_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_comment_id_actor_id_reac_24dc2de6_uniq UNIQUE (comment_id, actor_id, reaction, deleted_at);


--
-- Name: comment_reactions comment_reactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_pkey PRIMARY KEY (id);


--
-- Name: cycle_issues cycle_issue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issue_pkey PRIMARY KEY (id);


--
-- Name: cycle_issues cycle_issues_issue_id_cycle_id_deleted_at_93e8fecd_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issues_issue_id_cycle_id_deleted_at_93e8fecd_uniq UNIQUE (issue_id, cycle_id, deleted_at);


--
-- Name: cycles cycle_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycles
    ADD CONSTRAINT cycle_pkey PRIMARY KEY (id);


--
-- Name: cycle_user_properties cycle_user_properties_cycle_id_user_id_deleted_at_fbe00cf4_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_cycle_id_user_id_deleted_at_fbe00cf4_uniq UNIQUE (cycle_id, user_id, deleted_at);


--
-- Name: cycle_user_properties cycle_user_properties_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_pkey PRIMARY KEY (id);


--
-- Name: deploy_boards deploy_boards_anchor_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_anchor_key UNIQUE (anchor);


--
-- Name: deploy_boards deploy_boards_entity_name_entity_ident_800ce160_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_entity_name_entity_ident_800ce160_uniq UNIQUE (entity_name, entity_identifier, deleted_at);


--
-- Name: deploy_boards deploy_boards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_pkey PRIMARY KEY (id);


--
-- Name: description_versions description_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.description_versions
    ADD CONSTRAINT description_versions_pkey PRIMARY KEY (id);


--
-- Name: descriptions descriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.descriptions
    ADD CONSTRAINT descriptions_pkey PRIMARY KEY (id);


--
-- Name: device_sessions device_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_pkey PRIMARY KEY (id);


--
-- Name: devices devices_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices
    ADD CONSTRAINT devices_pkey PRIMARY KEY (id);


--
-- Name: draft_issue_assignees draft_issue_assignees_draft_issue_id_assignee__7cd49721_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_draft_issue_id_assignee__7cd49721_uniq UNIQUE (draft_issue_id, assignee_id, deleted_at);


--
-- Name: draft_issue_assignees draft_issue_assignees_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_pkey PRIMARY KEY (id);


--
-- Name: draft_issue_cycles draft_issue_cycles_draft_issue_id_cycle_id__e133e097_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_draft_issue_id_cycle_id__e133e097_uniq UNIQUE (draft_issue_id, cycle_id, deleted_at);


--
-- Name: draft_issue_cycles draft_issue_cycles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_pkey PRIMARY KEY (id);


--
-- Name: draft_issue_labels draft_issue_labels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_pkey PRIMARY KEY (id);


--
-- Name: draft_issue_modules draft_issue_modules_draft_issue_id_module_id_634e1f1a_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_draft_issue_id_module_id_634e1f1a_uniq UNIQUE (draft_issue_id, module_id, deleted_at);


--
-- Name: draft_issue_modules draft_issue_modules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_pkey PRIMARY KEY (id);


--
-- Name: draft_issues draft_issues_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_pkey PRIMARY KEY (id);


--
-- Name: email_notification_logs email_notification_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_notification_logs
    ADD CONSTRAINT email_notification_logs_pkey PRIMARY KEY (id);


--
-- Name: estimate_points estimate_points_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimate_points
    ADD CONSTRAINT estimate_points_pkey PRIMARY KEY (id);


--
-- Name: estimates estimates_name_project_id_deleted_at_41d66639_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimates
    ADD CONSTRAINT estimates_name_project_id_deleted_at_41d66639_uniq UNIQUE (name, project_id, deleted_at);


--
-- Name: estimates estimates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimates
    ADD CONSTRAINT estimates_pkey PRIMARY KEY (id);


--
-- Name: exporters exporters_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exporters
    ADD CONSTRAINT exporters_pkey PRIMARY KEY (id);


--
-- Name: exporters exporters_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exporters
    ADD CONSTRAINT exporters_token_key UNIQUE (token);


--
-- Name: file_assets file_asset_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_asset_pkey PRIMARY KEY (id);


--
-- Name: github_comment_syncs github_comment_syncs_issue_sync_id_comment_id_38c82e7b_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_issue_sync_id_comment_id_38c82e7b_uniq UNIQUE (issue_sync_id, comment_id);


--
-- Name: github_comment_syncs github_comment_syncs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_pkey PRIMARY KEY (id);


--
-- Name: github_issue_syncs github_issue_syncs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_pkey PRIMARY KEY (id);


--
-- Name: github_issue_syncs github_issue_syncs_repository_sync_id_issue_id_4b34427e_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_repository_sync_id_issue_id_4b34427e_uniq UNIQUE (repository_sync_id, issue_id);


--
-- Name: github_repositories github_repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repositories
    ADD CONSTRAINT github_repositories_pkey PRIMARY KEY (id);


--
-- Name: github_repository_syncs github_repository_syncs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_pkey PRIMARY KEY (id);


--
-- Name: github_repository_syncs github_repository_syncs_project_id_repository_id_0f3705e6_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_project_id_repository_id_0f3705e6_uniq UNIQUE (project_id, repository_id);


--
-- Name: github_repository_syncs github_repository_syncs_repository_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_repository_id_key UNIQUE (repository_id);


--
-- Name: importers importers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_pkey PRIMARY KEY (id);


--
-- Name: intake_issues inbox_issues_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_pkey PRIMARY KEY (id);


--
-- Name: intakes inboxes_name_project_id_deleted_at_95043f72_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intakes
    ADD CONSTRAINT inboxes_name_project_id_deleted_at_95043f72_uniq UNIQUE (name, project_id, deleted_at);


--
-- Name: intakes inboxes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intakes
    ADD CONSTRAINT inboxes_pkey PRIMARY KEY (id);


--
-- Name: instance_admins instance_admins_instance_id_user_id_2e80a466_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_admins
    ADD CONSTRAINT instance_admins_instance_id_user_id_2e80a466_uniq UNIQUE (instance_id, user_id);


--
-- Name: instance_admins instance_admins_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_admins
    ADD CONSTRAINT instance_admins_pkey PRIMARY KEY (id);


--
-- Name: instance_configurations instance_configurations_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_configurations
    ADD CONSTRAINT instance_configurations_key_key UNIQUE (key);


--
-- Name: instance_configurations instance_configurations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_configurations
    ADD CONSTRAINT instance_configurations_pkey PRIMARY KEY (id);


--
-- Name: instances instances_instance_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instances
    ADD CONSTRAINT instances_instance_id_key UNIQUE (instance_id);


--
-- Name: instances instances_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instances
    ADD CONSTRAINT instances_pkey PRIMARY KEY (id);


--
-- Name: integrations integrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_pkey PRIMARY KEY (id);


--
-- Name: integrations integrations_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_provider_key UNIQUE (provider);


--
-- Name: issue_activities issue_activity_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_pkey PRIMARY KEY (id);


--
-- Name: issue_assignees issue_assignee_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_pkey PRIMARY KEY (id);


--
-- Name: issue_assignees issue_assignees_issue_id_assignee_id_deleted_at_b2623a0e_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignees_issue_id_assignee_id_deleted_at_b2623a0e_uniq UNIQUE (issue_id, assignee_id, deleted_at);


--
-- Name: issue_attachments issue_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_attachments
    ADD CONSTRAINT issue_attachments_pkey PRIMARY KEY (id);


--
-- Name: issue_blockers issue_blocker_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_pkey PRIMARY KEY (id);


--
-- Name: issue_comments issue_comment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_pkey PRIMARY KEY (id);


--
-- Name: issue_comments issue_comments_description_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comments_description_id_key UNIQUE (description_id);


--
-- Name: issue_description_versions issue_description_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_versions_pkey PRIMARY KEY (id);


--
-- Name: issue_labels issue_label_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_pkey PRIMARY KEY (id);


--
-- Name: issue_links issue_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_links
    ADD CONSTRAINT issue_links_pkey PRIMARY KEY (id);


--
-- Name: issue_mentions issue_mentions_issue_id_mention_id_deleted_at_f6ecd6ed_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_issue_id_mention_id_deleted_at_f6ecd6ed_uniq UNIQUE (issue_id, mention_id, deleted_at);


--
-- Name: issue_mentions issue_mentions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_pkey PRIMARY KEY (id);


--
-- Name: issues issue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);


--
-- Name: project_user_properties issue_property_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_property_pkey PRIMARY KEY (id);


--
-- Name: issue_reactions issue_reactions_issue_id_actor_id_reacti_7da73ced_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_issue_id_actor_id_reacti_7da73ced_uniq UNIQUE (issue_id, actor_id, reaction, deleted_at);


--
-- Name: issue_reactions issue_reactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_pkey PRIMARY KEY (id);


--
-- Name: issue_relations issue_relations_issue_id_related_issue_i_cc724584_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_issue_id_related_issue_i_cc724584_uniq UNIQUE (issue_id, related_issue_id, deleted_at);


--
-- Name: issue_relations issue_relations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_pkey PRIMARY KEY (id);


--
-- Name: issue_sequences issue_sequence_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_sequences
    ADD CONSTRAINT issue_sequence_pkey PRIMARY KEY (id);


--
-- Name: issue_subscribers issue_subscribers_issue_id_subscriber_id_d_587dec1a_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_issue_id_subscriber_id_d_587dec1a_uniq UNIQUE (issue_id, subscriber_id, deleted_at);


--
-- Name: issue_subscribers issue_subscribers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_pkey PRIMARY KEY (id);


--
-- Name: issue_types issue_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_types
    ADD CONSTRAINT issue_types_pkey PRIMARY KEY (id);


--
-- Name: project_user_properties issue_user_properties_user_id_project_id_delet_2217dce5_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_user_properties_user_id_project_id_delet_2217dce5_uniq UNIQUE (user_id, project_id, deleted_at);


--
-- Name: issue_versions issue_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_pkey PRIMARY KEY (id);


--
-- Name: issue_views issue_views_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_views
    ADD CONSTRAINT issue_views_pkey PRIMARY KEY (id);


--
-- Name: issue_votes issue_votes_issue_id_actor_id_deleted_at_886f34e8_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_issue_id_actor_id_deleted_at_886f34e8_uniq UNIQUE (issue_id, actor_id, deleted_at);


--
-- Name: issue_votes issue_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_pkey PRIMARY KEY (id);


--
-- Name: labels label_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.labels
    ADD CONSTRAINT label_pkey PRIMARY KEY (id);


--
-- Name: module_issues module_issues_issue_id_module_id_deleted_at_f944f7c9_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_issue_id_module_id_deleted_at_f944f7c9_uniq UNIQUE (issue_id, module_id, deleted_at);


--
-- Name: module_issues module_issues_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_pkey PRIMARY KEY (id);


--
-- Name: module_links module_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_links
    ADD CONSTRAINT module_links_pkey PRIMARY KEY (id);


--
-- Name: module_members module_member_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_pkey PRIMARY KEY (id);


--
-- Name: module_members module_members_module_id_member_id_deleted_at_bb7a6f00_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_members_module_id_member_id_deleted_at_bb7a6f00_uniq UNIQUE (module_id, member_id, deleted_at);


--
-- Name: modules module_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT module_pkey PRIMARY KEY (id);


--
-- Name: module_user_properties module_user_properties_module_id_user_id_delete_3269582d_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_module_id_user_id_delete_3269582d_uniq UNIQUE (module_id, user_id, deleted_at);


--
-- Name: module_user_properties module_user_properties_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_pkey PRIMARY KEY (id);


--
-- Name: modules modules_name_project_id_deleted_at_328e2346_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT modules_name_project_id_deleted_at_328e2346_uniq UNIQUE (name, project_id, deleted_at);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: page_labels page_labels_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_labels
    ADD CONSTRAINT page_labels_pkey PRIMARY KEY (id);


--
-- Name: page_logs page_logs_page_id_transaction_9ab05334_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_logs
    ADD CONSTRAINT page_logs_page_id_transaction_9ab05334_uniq UNIQUE (page_id, transaction);


--
-- Name: page_logs page_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_logs
    ADD CONSTRAINT page_logs_pkey PRIMARY KEY (id);


--
-- Name: page_versions page_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_versions
    ADD CONSTRAINT page_versions_pkey PRIMARY KEY (id);


--
-- Name: pages pages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_pkey PRIMARY KEY (id);


--
-- Name: profiles profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_pkey PRIMARY KEY (id);


--
-- Name: profiles profiles_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_user_id_key UNIQUE (user_id);


--
-- Name: project_deploy_boards project_deploy_boards_anchor_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_anchor_key UNIQUE (anchor);


--
-- Name: project_deploy_boards project_deploy_boards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_pkey PRIMARY KEY (id);


--
-- Name: project_deploy_boards project_deploy_boards_project_id_anchor_893d365a_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_project_id_anchor_893d365a_uniq UNIQUE (project_id, anchor);


--
-- Name: project_identifiers project_identifier_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifier_pkey PRIMARY KEY (id);


--
-- Name: project_identifiers project_identifier_project_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifier_project_id_key UNIQUE (project_id);


--
-- Name: project_identifiers project_identifiers_name_workspace_id_deleted_at_d332e701_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifiers_name_workspace_id_deleted_at_d332e701_uniq UNIQUE (name, workspace_id, deleted_at);


--
-- Name: project_issue_types project_issue_types_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_pkey PRIMARY KEY (id);


--
-- Name: project_issue_types project_issue_types_project_id_issue_type_id_2287e5dc_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_project_id_issue_type_id_2287e5dc_uniq UNIQUE (project_id, issue_type_id, deleted_at);


--
-- Name: project_member_invites project_member_invite_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_member_invites
    ADD CONSTRAINT project_member_invite_pkey PRIMARY KEY (id);


--
-- Name: project_members project_member_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_member_pkey PRIMARY KEY (id);


--
-- Name: project_members project_members_project_id_member_id_deleted_at_0299122d_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_project_id_member_id_deleted_at_0299122d_uniq UNIQUE (project_id, member_id, deleted_at);


--
-- Name: project_pages project_pages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_pkey PRIMARY KEY (id);


--
-- Name: project_pages project_pages_project_id_page_id_deleted_at_7c80a40c_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_project_id_page_id_deleted_at_7c80a40c_uniq UNIQUE (project_id, page_id, deleted_at);


--
-- Name: projects project_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT project_pkey PRIMARY KEY (id);


--
-- Name: project_public_members project_public_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_pkey PRIMARY KEY (id);


--
-- Name: project_public_members project_public_members_project_id_member_id_del_9acd89b5_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_project_id_member_id_del_9acd89b5_uniq UNIQUE (project_id, member_id, deleted_at);


--
-- Name: project_webhooks project_webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_pkey PRIMARY KEY (id);


--
-- Name: project_webhooks project_webhooks_project_id_webhook_id_deleted_at_dcfdb35d_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_project_id_webhook_id_deleted_at_dcfdb35d_uniq UNIQUE (project_id, webhook_id, deleted_at);


--
-- Name: projects projects_identifier_workspace_id_deleted_at_7d2fa8c1_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_identifier_workspace_id_deleted_at_7d2fa8c1_uniq UNIQUE (identifier, workspace_id, deleted_at);


--
-- Name: projects projects_name_workspace_id_deleted_at_c7aa56f7_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_name_workspace_id_deleted_at_c7aa56f7_uniq UNIQUE (name, workspace_id, deleted_at);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (session_key);


--
-- Name: slack_project_syncs slack_project_syncs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_pkey PRIMARY KEY (id);


--
-- Name: slack_project_syncs slack_project_syncs_team_id_project_id_50a144a7_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_team_id_project_id_50a144a7_uniq UNIQUE (team_id, project_id);


--
-- Name: social_login_connections social_login_connection_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.social_login_connections
    ADD CONSTRAINT social_login_connection_pkey PRIMARY KEY (id);


--
-- Name: states state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.states
    ADD CONSTRAINT state_pkey PRIMARY KEY (id);


--
-- Name: states states_name_project_id_deleted_at_02f90488_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.states
    ADD CONSTRAINT states_name_project_id_deleted_at_02f90488_uniq UNIQUE (name, project_id, deleted_at);


--
-- Name: stickies stickies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stickies
    ADD CONSTRAINT stickies_pkey PRIMARY KEY (id);


--
-- Name: teams team_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT team_pkey PRIMARY KEY (id);


--
-- Name: teams teams_name_workspace_id_deleted_at_4b131aa2_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_name_workspace_id_deleted_at_4b131aa2_uniq UNIQUE (name, workspace_id, deleted_at);


--
-- Name: users user_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT user_email_key UNIQUE (email);


--
-- Name: user_favorites user_favorites_entity_type_user_id_enti_22b103ff_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_entity_type_user_id_enti_22b103ff_uniq UNIQUE (entity_type, user_id, entity_identifier, deleted_at);


--
-- Name: user_favorites user_favorites_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_pkey PRIMARY KEY (id);


--
-- Name: user_notification_preferences user_notification_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_preferences
    ADD CONSTRAINT user_notification_preferences_pkey PRIMARY KEY (id);


--
-- Name: users user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- Name: user_recent_visits user_recent_visits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_recent_visits
    ADD CONSTRAINT user_recent_visits_pkey PRIMARY KEY (id);


--
-- Name: users user_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT user_username_key UNIQUE (username);


--
-- Name: webhook_logs webhook_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_pkey PRIMARY KEY (id);


--
-- Name: webhooks webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);


--
-- Name: webhooks webhooks_workspace_id_url_deleted_at_ea7a1429_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_workspace_id_url_deleted_at_ea7a1429_uniq UNIQUE (workspace_id, url, deleted_at);


--
-- Name: workspace_home_preferences workspace_home_preferenc_workspace_id_user_id_key_75ea36d3_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_home_preferences
    ADD CONSTRAINT workspace_home_preferenc_workspace_id_user_id_key_75ea36d3_uniq UNIQUE (workspace_id, user_id, key, deleted_at);


--
-- Name: workspace_home_preferences workspace_home_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_home_preferences
    ADD CONSTRAINT workspace_home_preferences_pkey PRIMARY KEY (id);


--
-- Name: workspace_integrations workspace_integrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_pkey PRIMARY KEY (id);


--
-- Name: workspace_integrations workspace_integrations_workspace_id_integration_fa041c22_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_workspace_id_integration_fa041c22_uniq UNIQUE (workspace_id, integration_id);


--
-- Name: workspace_member_invites workspace_member_invite_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_member_invites
    ADD CONSTRAINT workspace_member_invite_pkey PRIMARY KEY (id);


--
-- Name: workspace_member_invites workspace_member_invites_email_workspace_id_delet_2f03573e_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_member_invites
    ADD CONSTRAINT workspace_member_invites_email_workspace_id_delet_2f03573e_uniq UNIQUE (email, workspace_id, deleted_at);


--
-- Name: workspace_members workspace_member_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_members
    ADD CONSTRAINT workspace_member_pkey PRIMARY KEY (id);


--
-- Name: workspace_members workspace_members_workspace_id_member_id_d_d7bfa872_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_members
    ADD CONSTRAINT workspace_members_workspace_id_member_id_d_d7bfa872_uniq UNIQUE (workspace_id, member_id, deleted_at);


--
-- Name: workspaces workspace_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspace_pkey PRIMARY KEY (id);


--
-- Name: workspaces workspace_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspace_slug_key UNIQUE (slug);


--
-- Name: workspace_themes workspace_themes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_themes
    ADD CONSTRAINT workspace_themes_pkey PRIMARY KEY (id);


--
-- Name: workspace_themes workspace_themes_workspace_id_name_deleted_at_b536ffd3_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_themes
    ADD CONSTRAINT workspace_themes_workspace_id_name_deleted_at_b536ffd3_uniq UNIQUE (workspace_id, name, deleted_at);


--
-- Name: workspace_user_links workspace_user_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_links
    ADD CONSTRAINT workspace_user_links_pkey PRIMARY KEY (id);


--
-- Name: workspace_user_preferences workspace_user_preferenc_workspace_id_user_id_key_79341493_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_preferences
    ADD CONSTRAINT workspace_user_preferenc_workspace_id_user_id_key_79341493_uniq UNIQUE (workspace_id, user_id, key, deleted_at);


--
-- Name: workspace_user_preferences workspace_user_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_preferences
    ADD CONSTRAINT workspace_user_preferences_pkey PRIMARY KEY (id);


--
-- Name: workspace_user_properties workspace_user_propertie_workspace_id_user_id_del_a7cf15bc_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_properties
    ADD CONSTRAINT workspace_user_propertie_workspace_id_user_id_del_a7cf15bc_uniq UNIQUE (workspace_id, user_id, deleted_at);


--
-- Name: workspace_user_properties workspace_user_properties_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_properties
    ADD CONSTRAINT workspace_user_properties_pkey PRIMARY KEY (id);


--
-- Name: accounts_user_id_7f1e1f1e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX accounts_user_id_7f1e1f1e ON public.accounts USING btree (user_id);


--
-- Name: analytic_views_created_by_id_1b3ca0a9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX analytic_views_created_by_id_1b3ca0a9 ON public.analytic_views USING btree (created_by_id);


--
-- Name: analytic_views_updated_by_id_b6d827e1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX analytic_views_updated_by_id_b6d827e1 ON public.analytic_views USING btree (updated_by_id);


--
-- Name: analytic_views_workspace_id_ca6e5c0b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX analytic_views_workspace_id_ca6e5c0b ON public.analytic_views USING btree (workspace_id);


--
-- Name: api_activity_logs_created_by_id_7f5c4ca8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_activity_logs_created_by_id_7f5c4ca8 ON public.api_activity_logs USING btree (created_by_id);


--
-- Name: api_activity_logs_updated_by_id_9ba0d417; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_activity_logs_updated_by_id_9ba0d417 ON public.api_activity_logs USING btree (updated_by_id);


--
-- Name: api_tokens_created_by_id_441e3d24; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_tokens_created_by_id_441e3d24 ON public.api_tokens USING btree (created_by_id);


--
-- Name: api_tokens_token_6211101f_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_tokens_token_6211101f_like ON public.api_tokens USING btree (token varchar_pattern_ops);


--
-- Name: api_tokens_updated_by_id_bcd544cf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_tokens_updated_by_id_bcd544cf ON public.api_tokens USING btree (updated_by_id);


--
-- Name: api_tokens_user_id_2db24e1c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_tokens_user_id_2db24e1c ON public.api_tokens USING btree (user_id);


--
-- Name: api_tokens_workspace_id_6791c7bd; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX api_tokens_workspace_id_6791c7bd ON public.api_tokens USING btree (workspace_id);


--
-- Name: asset_asset_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX asset_asset_idx ON public.file_assets USING btree (asset);


--
-- Name: asset_entity_identifier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX asset_entity_identifier_idx ON public.file_assets USING btree (entity_identifier);


--
-- Name: asset_entity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX asset_entity_idx ON public.file_assets USING btree (entity_type, entity_identifier);


--
-- Name: asset_entity_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX asset_entity_type_idx ON public.file_assets USING btree (entity_type);


--
-- Name: changelogs_created_by_id_16dd944a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changelogs_created_by_id_16dd944a ON public.changelogs USING btree (created_by_id);


--
-- Name: changelogs_updated_by_id_e0989861; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX changelogs_updated_by_id_e0989861 ON public.changelogs USING btree (updated_by_id);


--
-- Name: comment_reaction_unique_comment_actor_reaction_when_deleted_at_; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX comment_reaction_unique_comment_actor_reaction_when_deleted_at_ ON public.comment_reactions USING btree (comment_id, actor_id, reaction) WHERE (deleted_at IS NULL);


--
-- Name: comment_reactions_actor_id_21219e9c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reactions_actor_id_21219e9c ON public.comment_reactions USING btree (actor_id);


--
-- Name: comment_reactions_comment_id_87c59446; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reactions_comment_id_87c59446 ON public.comment_reactions USING btree (comment_id);


--
-- Name: comment_reactions_created_by_id_9aeb43c4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reactions_created_by_id_9aeb43c4 ON public.comment_reactions USING btree (created_by_id);


--
-- Name: comment_reactions_project_id_ab9114b4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reactions_project_id_ab9114b4 ON public.comment_reactions USING btree (project_id);


--
-- Name: comment_reactions_updated_by_id_c74c9bbd; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reactions_updated_by_id_c74c9bbd ON public.comment_reactions USING btree (updated_by_id);


--
-- Name: comment_reactions_workspace_id_b614ca4f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_reactions_workspace_id_b614ca4f ON public.comment_reactions USING btree (workspace_id);


--
-- Name: cycle_created_by_id_78e43b79; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_created_by_id_78e43b79 ON public.cycles USING btree (created_by_id);


--
-- Name: cycle_issue_created_by_id_30b27539; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_issue_created_by_id_30b27539 ON public.cycle_issues USING btree (created_by_id);


--
-- Name: cycle_issue_cycle_id_ec681215; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_issue_cycle_id_ec681215 ON public.cycle_issues USING btree (cycle_id);


--
-- Name: cycle_issue_project_id_6ad3257a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_issue_project_id_6ad3257a ON public.cycle_issues USING btree (project_id);


--
-- Name: cycle_issue_updated_by_id_cb4516f2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_issue_updated_by_id_cb4516f2 ON public.cycle_issues USING btree (updated_by_id);


--
-- Name: cycle_issue_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX cycle_issue_when_deleted_at_null ON public.cycle_issues USING btree (cycle_id, issue_id) WHERE (deleted_at IS NULL);


--
-- Name: cycle_issue_workspace_id_1d77330e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_issue_workspace_id_1d77330e ON public.cycle_issues USING btree (workspace_id);


--
-- Name: cycle_issues_issue_id_2d5ac97f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_issues_issue_id_2d5ac97f ON public.cycle_issues USING btree (issue_id);


--
-- Name: cycle_owned_by_id_5456a4d1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_owned_by_id_5456a4d1 ON public.cycles USING btree (owned_by_id);


--
-- Name: cycle_project_id_0b590349; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_project_id_0b590349 ON public.cycles USING btree (project_id);


--
-- Name: cycle_updated_by_id_93baee43; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_updated_by_id_93baee43 ON public.cycles USING btree (updated_by_id);


--
-- Name: cycle_user_properties_created_by_id_501f371c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_user_properties_created_by_id_501f371c ON public.cycle_user_properties USING btree (created_by_id);


--
-- Name: cycle_user_properties_cycle_id_1f8bdf35; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_user_properties_cycle_id_1f8bdf35 ON public.cycle_user_properties USING btree (cycle_id);


--
-- Name: cycle_user_properties_project_id_4efc0f07; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_user_properties_project_id_4efc0f07 ON public.cycle_user_properties USING btree (project_id);


--
-- Name: cycle_user_properties_unique_cycle_user_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX cycle_user_properties_unique_cycle_user_when_deleted_at_null ON public.cycle_user_properties USING btree (cycle_id, user_id) WHERE (deleted_at IS NULL);


--
-- Name: cycle_user_properties_updated_by_id_1b5ac27b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_user_properties_updated_by_id_1b5ac27b ON public.cycle_user_properties USING btree (updated_by_id);


--
-- Name: cycle_user_properties_user_id_9e9ef97d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_user_properties_user_id_9e9ef97d ON public.cycle_user_properties USING btree (user_id);


--
-- Name: cycle_user_properties_workspace_id_62d65d71; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_user_properties_workspace_id_62d65d71 ON public.cycle_user_properties USING btree (workspace_id);


--
-- Name: cycle_workspace_id_a199e8e1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX cycle_workspace_id_a199e8e1 ON public.cycles USING btree (workspace_id);


--
-- Name: deploy_board_unique_entity_name_entity_identifier_when_deleted_; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX deploy_board_unique_entity_name_entity_identifier_when_deleted_ ON public.deploy_boards USING btree (entity_name, entity_identifier) WHERE (deleted_at IS NULL);


--
-- Name: deploy_boards_anchor_fe87f323_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX deploy_boards_anchor_fe87f323_like ON public.deploy_boards USING btree (anchor varchar_pattern_ops);


--
-- Name: deploy_boards_created_by_id_149dff93; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX deploy_boards_created_by_id_149dff93 ON public.deploy_boards USING btree (created_by_id);


--
-- Name: deploy_boards_inbox_id_ebc13d44; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX deploy_boards_inbox_id_ebc13d44 ON public.deploy_boards USING btree (intake_id);


--
-- Name: deploy_boards_project_id_cfc792a1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX deploy_boards_project_id_cfc792a1 ON public.deploy_boards USING btree (project_id);


--
-- Name: deploy_boards_updated_by_id_db7ae24f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX deploy_boards_updated_by_id_db7ae24f ON public.deploy_boards USING btree (updated_by_id);


--
-- Name: deploy_boards_workspace_id_fcf03158; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX deploy_boards_workspace_id_fcf03158 ON public.deploy_boards USING btree (workspace_id);


--
-- Name: description_versions_created_by_id_6633a3de; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX description_versions_created_by_id_6633a3de ON public.description_versions USING btree (created_by_id);


--
-- Name: description_versions_description_id_dc7f19b6; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX description_versions_description_id_dc7f19b6 ON public.description_versions USING btree (description_id);


--
-- Name: description_versions_project_id_1a6c9aa9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX description_versions_project_id_1a6c9aa9 ON public.description_versions USING btree (project_id);


--
-- Name: description_versions_updated_by_id_8b5179ae; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX description_versions_updated_by_id_8b5179ae ON public.description_versions USING btree (updated_by_id);


--
-- Name: description_versions_workspace_id_52857186; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX description_versions_workspace_id_52857186 ON public.description_versions USING btree (workspace_id);


--
-- Name: descriptions_created_by_id_b88ab399; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX descriptions_created_by_id_b88ab399 ON public.descriptions USING btree (created_by_id);


--
-- Name: descriptions_project_id_8f46180b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX descriptions_project_id_8f46180b ON public.descriptions USING btree (project_id);


--
-- Name: descriptions_updated_by_id_af519c4d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX descriptions_updated_by_id_af519c4d ON public.descriptions USING btree (updated_by_id);


--
-- Name: descriptions_workspace_id_767279bf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX descriptions_workspace_id_767279bf ON public.descriptions USING btree (workspace_id);


--
-- Name: device_sessions_created_by_id_920a3bd5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_sessions_created_by_id_920a3bd5 ON public.device_sessions USING btree (created_by_id);


--
-- Name: device_sessions_device_id_a42b2ada; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_sessions_device_id_a42b2ada ON public.device_sessions USING btree (device_id);


--
-- Name: device_sessions_session_id_5382b02b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_sessions_session_id_5382b02b ON public.device_sessions USING btree (session_id);


--
-- Name: device_sessions_session_id_5382b02b_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_sessions_session_id_5382b02b_like ON public.device_sessions USING btree (session_id varchar_pattern_ops);


--
-- Name: device_sessions_updated_by_id_d0bd0c76; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX device_sessions_updated_by_id_d0bd0c76 ON public.device_sessions USING btree (updated_by_id);


--
-- Name: devices_created_by_id_410a755b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_created_by_id_410a755b ON public.devices USING btree (created_by_id);


--
-- Name: devices_updated_by_id_ee20dc3c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_updated_by_id_ee20dc3c ON public.devices USING btree (updated_by_id);


--
-- Name: devices_user_id_9a5cca49; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX devices_user_id_9a5cca49 ON public.devices USING btree (user_id);


--
-- Name: draft_issue_assignee_unique_issue_assignee_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX draft_issue_assignee_unique_issue_assignee_when_deleted_at_null ON public.draft_issue_assignees USING btree (draft_issue_id, assignee_id) WHERE (deleted_at IS NULL);


--
-- Name: draft_issue_assignees_assignee_id_9cc52f9d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_assignees_assignee_id_9cc52f9d ON public.draft_issue_assignees USING btree (assignee_id);


--
-- Name: draft_issue_assignees_created_by_id_c25d4bde; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_assignees_created_by_id_c25d4bde ON public.draft_issue_assignees USING btree (created_by_id);


--
-- Name: draft_issue_assignees_draft_issue_id_70827be2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_assignees_draft_issue_id_70827be2 ON public.draft_issue_assignees USING btree (draft_issue_id);


--
-- Name: draft_issue_assignees_project_id_c87dd571; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_assignees_project_id_c87dd571 ON public.draft_issue_assignees USING btree (project_id);


--
-- Name: draft_issue_assignees_updated_by_id_16dbb5e0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_assignees_updated_by_id_16dbb5e0 ON public.draft_issue_assignees USING btree (updated_by_id);


--
-- Name: draft_issue_assignees_workspace_id_e28a98e9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_assignees_workspace_id_e28a98e9 ON public.draft_issue_assignees USING btree (workspace_id);


--
-- Name: draft_issue_cycle_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX draft_issue_cycle_when_deleted_at_null ON public.draft_issue_cycles USING btree (draft_issue_id, cycle_id) WHERE (deleted_at IS NULL);


--
-- Name: draft_issue_cycles_created_by_id_e56335c8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_cycles_created_by_id_e56335c8 ON public.draft_issue_cycles USING btree (created_by_id);


--
-- Name: draft_issue_cycles_cycle_id_b214e11f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_cycles_cycle_id_b214e11f ON public.draft_issue_cycles USING btree (cycle_id);


--
-- Name: draft_issue_cycles_draft_issue_id_ed45e8a2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_cycles_draft_issue_id_ed45e8a2 ON public.draft_issue_cycles USING btree (draft_issue_id);


--
-- Name: draft_issue_cycles_project_id_dc5d1ff6; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_cycles_project_id_dc5d1ff6 ON public.draft_issue_cycles USING btree (project_id);


--
-- Name: draft_issue_cycles_updated_by_id_518a23ab; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_cycles_updated_by_id_518a23ab ON public.draft_issue_cycles USING btree (updated_by_id);


--
-- Name: draft_issue_cycles_workspace_id_4fd0aa0c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_cycles_workspace_id_4fd0aa0c ON public.draft_issue_cycles USING btree (workspace_id);


--
-- Name: draft_issue_labels_created_by_id_88217eef; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_labels_created_by_id_88217eef ON public.draft_issue_labels USING btree (created_by_id);


--
-- Name: draft_issue_labels_draft_issue_id_339d4c2b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_labels_draft_issue_id_339d4c2b ON public.draft_issue_labels USING btree (draft_issue_id);


--
-- Name: draft_issue_labels_label_id_b9b001a5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_labels_label_id_b9b001a5 ON public.draft_issue_labels USING btree (label_id);


--
-- Name: draft_issue_labels_project_id_16f9ba0a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_labels_project_id_16f9ba0a ON public.draft_issue_labels USING btree (project_id);


--
-- Name: draft_issue_labels_updated_by_id_edac537c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_labels_updated_by_id_edac537c ON public.draft_issue_labels USING btree (updated_by_id);


--
-- Name: draft_issue_labels_workspace_id_489a9873; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_labels_workspace_id_489a9873 ON public.draft_issue_labels USING btree (workspace_id);


--
-- Name: draft_issue_modules_created_by_id_95ec4247; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_modules_created_by_id_95ec4247 ON public.draft_issue_modules USING btree (created_by_id);


--
-- Name: draft_issue_modules_draft_issue_id_eb470383; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_modules_draft_issue_id_eb470383 ON public.draft_issue_modules USING btree (draft_issue_id);


--
-- Name: draft_issue_modules_module_id_4d3f477a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_modules_module_id_4d3f477a ON public.draft_issue_modules USING btree (module_id);


--
-- Name: draft_issue_modules_project_id_c32eadab; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_modules_project_id_c32eadab ON public.draft_issue_modules USING btree (project_id);


--
-- Name: draft_issue_modules_updated_by_id_18548965; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_modules_updated_by_id_18548965 ON public.draft_issue_modules USING btree (updated_by_id);


--
-- Name: draft_issue_modules_workspace_id_536c335a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issue_modules_workspace_id_536c335a ON public.draft_issue_modules USING btree (workspace_id);


--
-- Name: draft_issues_created_by_id_aedba72a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_created_by_id_aedba72a ON public.draft_issues USING btree (created_by_id);


--
-- Name: draft_issues_estimate_point_id_9e333189; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_estimate_point_id_9e333189 ON public.draft_issues USING btree (estimate_point_id);


--
-- Name: draft_issues_parent_id_eee6ec32; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_parent_id_eee6ec32 ON public.draft_issues USING btree (parent_id);


--
-- Name: draft_issues_project_id_784a560c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_project_id_784a560c ON public.draft_issues USING btree (project_id);


--
-- Name: draft_issues_state_id_94f28f5a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_state_id_94f28f5a ON public.draft_issues USING btree (state_id);


--
-- Name: draft_issues_type_id_7a62fe34; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_type_id_7a62fe34 ON public.draft_issues USING btree (type_id);


--
-- Name: draft_issues_updated_by_id_1ca3cd4e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_updated_by_id_1ca3cd4e ON public.draft_issues USING btree (updated_by_id);


--
-- Name: draft_issues_workspace_id_9d8512c8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX draft_issues_workspace_id_9d8512c8 ON public.draft_issues USING btree (workspace_id);


--
-- Name: email_notification_logs_created_by_id_6faff587; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX email_notification_logs_created_by_id_6faff587 ON public.email_notification_logs USING btree (created_by_id);


--
-- Name: email_notification_logs_receiver_id_7c7d2e13; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX email_notification_logs_receiver_id_7c7d2e13 ON public.email_notification_logs USING btree (receiver_id);


--
-- Name: email_notification_logs_triggered_by_id_b551e727; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX email_notification_logs_triggered_by_id_b551e727 ON public.email_notification_logs USING btree (triggered_by_id);


--
-- Name: email_notification_logs_updated_by_id_5d99c798; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX email_notification_logs_updated_by_id_5d99c798 ON public.email_notification_logs USING btree (updated_by_id);


--
-- Name: estimate_points_created_by_id_d1b04bd9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimate_points_created_by_id_d1b04bd9 ON public.estimate_points USING btree (created_by_id);


--
-- Name: estimate_points_estimate_id_4b4cb706; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimate_points_estimate_id_4b4cb706 ON public.estimate_points USING btree (estimate_id);


--
-- Name: estimate_points_project_id_ba9bcb2c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimate_points_project_id_ba9bcb2c ON public.estimate_points USING btree (project_id);


--
-- Name: estimate_points_updated_by_id_a1da94e1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimate_points_updated_by_id_a1da94e1 ON public.estimate_points USING btree (updated_by_id);


--
-- Name: estimate_points_workspace_id_96fc4f92; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimate_points_workspace_id_96fc4f92 ON public.estimate_points USING btree (workspace_id);


--
-- Name: estimate_unique_name_project_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX estimate_unique_name_project_when_deleted_at_null ON public.estimates USING btree (name, project_id) WHERE (deleted_at IS NULL);


--
-- Name: estimates_created_by_id_7e401493; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimates_created_by_id_7e401493 ON public.estimates USING btree (created_by_id);


--
-- Name: estimates_project_id_7f195a41; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimates_project_id_7f195a41 ON public.estimates USING btree (project_id);


--
-- Name: estimates_updated_by_id_b3fcfb1d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimates_updated_by_id_b3fcfb1d ON public.estimates USING btree (updated_by_id);


--
-- Name: estimates_workspace_id_718811eb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX estimates_workspace_id_718811eb ON public.estimates USING btree (workspace_id);


--
-- Name: exporters_created_by_id_44e1d9b3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exporters_created_by_id_44e1d9b3 ON public.exporters USING btree (created_by_id);


--
-- Name: exporters_initiated_by_id_d51f7552; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exporters_initiated_by_id_d51f7552 ON public.exporters USING btree (initiated_by_id);


--
-- Name: exporters_token_c774aeeb_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exporters_token_c774aeeb_like ON public.exporters USING btree (token varchar_pattern_ops);


--
-- Name: exporters_updated_by_id_d2572861; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exporters_updated_by_id_d2572861 ON public.exporters USING btree (updated_by_id);


--
-- Name: exporters_workspace_id_11a04317; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX exporters_workspace_id_11a04317 ON public.exporters USING btree (workspace_id);


--
-- Name: fav_entity_identifier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fav_entity_identifier_idx ON public.user_favorites USING btree (entity_identifier);


--
-- Name: fav_entity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fav_entity_idx ON public.user_favorites USING btree (entity_type, entity_identifier);


--
-- Name: fav_entity_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX fav_entity_type_idx ON public.user_favorites USING btree (entity_type);


--
-- Name: file_asset_created_by_id_966942a0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_asset_created_by_id_966942a0 ON public.file_assets USING btree (created_by_id);


--
-- Name: file_asset_updated_by_id_d6aaf4f0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_asset_updated_by_id_d6aaf4f0 ON public.file_assets USING btree (updated_by_id);


--
-- Name: file_assets_comment_id_35d4ecaf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_comment_id_35d4ecaf ON public.file_assets USING btree (comment_id);


--
-- Name: file_assets_draft_issue_id_52633145; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_draft_issue_id_52633145 ON public.file_assets USING btree (draft_issue_id);


--
-- Name: file_assets_issue_id_cfe87d6c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_issue_id_cfe87d6c ON public.file_assets USING btree (issue_id);


--
-- Name: file_assets_page_id_64c753d1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_page_id_64c753d1 ON public.file_assets USING btree (page_id);


--
-- Name: file_assets_project_id_ebd5c0d8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_project_id_ebd5c0d8 ON public.file_assets USING btree (project_id);


--
-- Name: file_assets_user_id_ce1818dc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_user_id_ce1818dc ON public.file_assets USING btree (user_id);


--
-- Name: file_assets_workspace_id_fa50b9c5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX file_assets_workspace_id_fa50b9c5 ON public.file_assets USING btree (workspace_id);


--
-- Name: github_comment_syncs_comment_id_6feec6d1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_comment_syncs_comment_id_6feec6d1 ON public.github_comment_syncs USING btree (comment_id);


--
-- Name: github_comment_syncs_created_by_id_b1ef2517; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_comment_syncs_created_by_id_b1ef2517 ON public.github_comment_syncs USING btree (created_by_id);


--
-- Name: github_comment_syncs_issue_sync_id_5e738eb5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_comment_syncs_issue_sync_id_5e738eb5 ON public.github_comment_syncs USING btree (issue_sync_id);


--
-- Name: github_comment_syncs_project_id_6d199ace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_comment_syncs_project_id_6d199ace ON public.github_comment_syncs USING btree (project_id);


--
-- Name: github_comment_syncs_updated_by_id_bb05c066; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_comment_syncs_updated_by_id_bb05c066 ON public.github_comment_syncs USING btree (updated_by_id);


--
-- Name: github_comment_syncs_workspace_id_b54528c8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_comment_syncs_workspace_id_b54528c8 ON public.github_comment_syncs USING btree (workspace_id);


--
-- Name: github_issue_syncs_created_by_id_d02b7c56; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_issue_syncs_created_by_id_d02b7c56 ON public.github_issue_syncs USING btree (created_by_id);


--
-- Name: github_issue_syncs_issue_id_450cb083; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_issue_syncs_issue_id_450cb083 ON public.github_issue_syncs USING btree (issue_id);


--
-- Name: github_issue_syncs_project_id_4609ad0c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_issue_syncs_project_id_4609ad0c ON public.github_issue_syncs USING btree (project_id);


--
-- Name: github_issue_syncs_repository_sync_id_ba0d4de4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_issue_syncs_repository_sync_id_ba0d4de4 ON public.github_issue_syncs USING btree (repository_sync_id);


--
-- Name: github_issue_syncs_updated_by_id_e9cd6f86; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_issue_syncs_updated_by_id_e9cd6f86 ON public.github_issue_syncs USING btree (updated_by_id);


--
-- Name: github_issue_syncs_workspace_id_eae020ad; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_issue_syncs_workspace_id_eae020ad ON public.github_issue_syncs USING btree (workspace_id);


--
-- Name: github_repositories_created_by_id_104fa685; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repositories_created_by_id_104fa685 ON public.github_repositories USING btree (created_by_id);


--
-- Name: github_repositories_project_id_65c546bb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repositories_project_id_65c546bb ON public.github_repositories USING btree (project_id);


--
-- Name: github_repositories_updated_by_id_8aa4d772; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repositories_updated_by_id_8aa4d772 ON public.github_repositories USING btree (updated_by_id);


--
-- Name: github_repositories_workspace_id_c4de7326; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repositories_workspace_id_c4de7326 ON public.github_repositories USING btree (workspace_id);


--
-- Name: github_repository_syncs_actor_id_1fa689fe; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_actor_id_1fa689fe ON public.github_repository_syncs USING btree (actor_id);


--
-- Name: github_repository_syncs_created_by_id_0df94495; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_created_by_id_0df94495 ON public.github_repository_syncs USING btree (created_by_id);


--
-- Name: github_repository_syncs_label_id_eb1e9bd7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_label_id_eb1e9bd7 ON public.github_repository_syncs USING btree (label_id);


--
-- Name: github_repository_syncs_project_id_e7e8291e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_project_id_e7e8291e ON public.github_repository_syncs USING btree (project_id);


--
-- Name: github_repository_syncs_updated_by_id_07e9d065; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_updated_by_id_07e9d065 ON public.github_repository_syncs USING btree (updated_by_id);


--
-- Name: github_repository_syncs_workspace_id_4a22a8b8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_workspace_id_4a22a8b8 ON public.github_repository_syncs USING btree (workspace_id);


--
-- Name: github_repository_syncs_workspace_integration_id_62858398; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX github_repository_syncs_workspace_integration_id_62858398 ON public.github_repository_syncs USING btree (workspace_integration_id);


--
-- Name: importers_created_by_id_7dd06433; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX importers_created_by_id_7dd06433 ON public.importers USING btree (created_by_id);


--
-- Name: importers_initiated_by_id_3cddbd23; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX importers_initiated_by_id_3cddbd23 ON public.importers USING btree (initiated_by_id);


--
-- Name: importers_project_id_1f8b43ef; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX importers_project_id_1f8b43ef ON public.importers USING btree (project_id);


--
-- Name: importers_token_id_c951e89f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX importers_token_id_c951e89f ON public.importers USING btree (token_id);


--
-- Name: importers_updated_by_id_3915139e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX importers_updated_by_id_3915139e ON public.importers USING btree (updated_by_id);


--
-- Name: importers_workspace_id_795b8985; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX importers_workspace_id_795b8985 ON public.importers USING btree (workspace_id);


--
-- Name: inbox_issues_created_by_id_483bce13; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_created_by_id_483bce13 ON public.intake_issues USING btree (created_by_id);


--
-- Name: inbox_issues_duplicate_to_id_6cb8d961; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_duplicate_to_id_6cb8d961 ON public.intake_issues USING btree (duplicate_to_id);


--
-- Name: inbox_issues_inbox_id_444b05b9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_inbox_id_444b05b9 ON public.intake_issues USING btree (intake_id);


--
-- Name: inbox_issues_issue_id_7d74b224; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_issue_id_7d74b224 ON public.intake_issues USING btree (issue_id);


--
-- Name: inbox_issues_project_id_5117a70b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_project_id_5117a70b ON public.intake_issues USING btree (project_id);


--
-- Name: inbox_issues_updated_by_id_d1b2b70f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_updated_by_id_d1b2b70f ON public.intake_issues USING btree (updated_by_id);


--
-- Name: inbox_issues_workspace_id_4a61a7bd; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inbox_issues_workspace_id_4a61a7bd ON public.intake_issues USING btree (workspace_id);


--
-- Name: inboxes_created_by_id_9f1cf5ec; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inboxes_created_by_id_9f1cf5ec ON public.intakes USING btree (created_by_id);


--
-- Name: inboxes_project_id_a0135c66; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inboxes_project_id_a0135c66 ON public.intakes USING btree (project_id);


--
-- Name: inboxes_updated_by_id_69b7b3ae; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inboxes_updated_by_id_69b7b3ae ON public.intakes USING btree (updated_by_id);


--
-- Name: inboxes_workspace_id_d6178865; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX inboxes_workspace_id_d6178865 ON public.intakes USING btree (workspace_id);


--
-- Name: instance_admins_created_by_id_7f4e03b4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_admins_created_by_id_7f4e03b4 ON public.instance_admins USING btree (created_by_id);


--
-- Name: instance_admins_instance_id_66d1ba73; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_admins_instance_id_66d1ba73 ON public.instance_admins USING btree (instance_id);


--
-- Name: instance_admins_updated_by_id_b7800403; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_admins_updated_by_id_b7800403 ON public.instance_admins USING btree (updated_by_id);


--
-- Name: instance_admins_user_id_cc6e9b62; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_admins_user_id_cc6e9b62 ON public.instance_admins USING btree (user_id);


--
-- Name: instance_configurations_created_by_id_e683f3e5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_configurations_created_by_id_e683f3e5 ON public.instance_configurations USING btree (created_by_id);


--
-- Name: instance_configurations_key_3eb64d36_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_configurations_key_3eb64d36_like ON public.instance_configurations USING btree (key varchar_pattern_ops);


--
-- Name: instance_configurations_updated_by_id_f0d7542e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instance_configurations_updated_by_id_f0d7542e ON public.instance_configurations USING btree (updated_by_id);


--
-- Name: instances_created_by_id_c76e92ef; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instances_created_by_id_c76e92ef ON public.instances USING btree (created_by_id);


--
-- Name: instances_instance_id_cf688621_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instances_instance_id_cf688621_like ON public.instances USING btree (instance_id varchar_pattern_ops);


--
-- Name: instances_updated_by_id_cce8fcdf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX instances_updated_by_id_cce8fcdf ON public.instances USING btree (updated_by_id);


--
-- Name: intake_unique_name_project_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX intake_unique_name_project_when_deleted_at_null ON public.intakes USING btree (name, project_id) WHERE (deleted_at IS NULL);


--
-- Name: integrations_created_by_id_0b6edd52; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX integrations_created_by_id_0b6edd52 ON public.integrations USING btree (created_by_id);


--
-- Name: integrations_provider_6537a106_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX integrations_provider_6537a106_like ON public.integrations USING btree (provider varchar_pattern_ops);


--
-- Name: integrations_updated_by_id_d6d00d15; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX integrations_updated_by_id_d6d00d15 ON public.integrations USING btree (updated_by_id);


--
-- Name: issue_activity_actor_id_52fdd42d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_actor_id_52fdd42d ON public.issue_activities USING btree (actor_id);


--
-- Name: issue_activity_created_by_id_49516e3d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_created_by_id_49516e3d ON public.issue_activities USING btree (created_by_id);


--
-- Name: issue_activity_issue_comment_id_701f3c3c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_issue_comment_id_701f3c3c ON public.issue_activities USING btree (issue_comment_id);


--
-- Name: issue_activity_issue_id_807fbde4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_issue_id_807fbde4 ON public.issue_activities USING btree (issue_id);


--
-- Name: issue_activity_project_id_d0ac2ccf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_project_id_d0ac2ccf ON public.issue_activities USING btree (project_id);


--
-- Name: issue_activity_updated_by_id_0075f9bd; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_updated_by_id_0075f9bd ON public.issue_activities USING btree (updated_by_id);


--
-- Name: issue_activity_workspace_id_65acaf73; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_activity_workspace_id_65acaf73 ON public.issue_activities USING btree (workspace_id);


--
-- Name: issue_assignee_assignee_id_50f5c04e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_assignee_assignee_id_50f5c04e ON public.issue_assignees USING btree (assignee_id);


--
-- Name: issue_assignee_created_by_id_f693d43b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_assignee_created_by_id_f693d43b ON public.issue_assignees USING btree (created_by_id);


--
-- Name: issue_assignee_issue_id_72da08db; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_assignee_issue_id_72da08db ON public.issue_assignees USING btree (issue_id);


--
-- Name: issue_assignee_project_id_61c18bf2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_assignee_project_id_61c18bf2 ON public.issue_assignees USING btree (project_id);


--
-- Name: issue_assignee_unique_issue_assignee_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_assignee_unique_issue_assignee_when_deleted_at_null ON public.issue_assignees USING btree (issue_id, assignee_id) WHERE (deleted_at IS NULL);


--
-- Name: issue_assignee_updated_by_id_c54088aa; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_assignee_updated_by_id_c54088aa ON public.issue_assignees USING btree (updated_by_id);


--
-- Name: issue_assignee_workspace_id_9aad55b7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_assignee_workspace_id_9aad55b7 ON public.issue_assignees USING btree (workspace_id);


--
-- Name: issue_attachments_created_by_id_87be05bb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_attachments_created_by_id_87be05bb ON public.issue_attachments USING btree (created_by_id);


--
-- Name: issue_attachments_issue_id_0faf88bf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_attachments_issue_id_0faf88bf ON public.issue_attachments USING btree (issue_id);


--
-- Name: issue_attachments_project_id_a95fe706; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_attachments_project_id_a95fe706 ON public.issue_attachments USING btree (project_id);


--
-- Name: issue_attachments_updated_by_id_47dceec1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_attachments_updated_by_id_47dceec1 ON public.issue_attachments USING btree (updated_by_id);


--
-- Name: issue_attachments_workspace_id_c456a532; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_attachments_workspace_id_c456a532 ON public.issue_attachments USING btree (workspace_id);


--
-- Name: issue_blocker_block_id_5d15a701; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_blocker_block_id_5d15a701 ON public.issue_blockers USING btree (block_id);


--
-- Name: issue_blocker_blocked_by_id_a138af71; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_blocker_blocked_by_id_a138af71 ON public.issue_blockers USING btree (blocked_by_id);


--
-- Name: issue_blocker_created_by_id_0d19f6ea; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_blocker_created_by_id_0d19f6ea ON public.issue_blockers USING btree (created_by_id);


--
-- Name: issue_blocker_project_id_380bd100; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_blocker_project_id_380bd100 ON public.issue_blockers USING btree (project_id);


--
-- Name: issue_blocker_updated_by_id_4af87d63; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_blocker_updated_by_id_4af87d63 ON public.issue_blockers USING btree (updated_by_id);


--
-- Name: issue_blocker_workspace_id_419a1c71; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_blocker_workspace_id_419a1c71 ON public.issue_blockers USING btree (workspace_id);


--
-- Name: issue_comment_actor_id_d312315b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comment_actor_id_d312315b ON public.issue_comments USING btree (actor_id);


--
-- Name: issue_comment_created_by_id_0765f239; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comment_created_by_id_0765f239 ON public.issue_comments USING btree (created_by_id);


--
-- Name: issue_comment_issue_id_d0195e35; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comment_issue_id_d0195e35 ON public.issue_comments USING btree (issue_id);


--
-- Name: issue_comment_project_id_db37c105; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comment_project_id_db37c105 ON public.issue_comments USING btree (project_id);


--
-- Name: issue_comment_updated_by_id_96cfb86e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comment_updated_by_id_96cfb86e ON public.issue_comments USING btree (updated_by_id);


--
-- Name: issue_comment_workspace_id_3f7969ec; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comment_workspace_id_3f7969ec ON public.issue_comments USING btree (workspace_id);


--
-- Name: issue_comments_parent_id_d8db10b1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_comments_parent_id_d8db10b1 ON public.issue_comments USING btree (parent_id);


--
-- Name: issue_created_by_id_8f0ae62b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_created_by_id_8f0ae62b ON public.issues USING btree (created_by_id);


--
-- Name: issue_description_versions_created_by_id_3f7e62a1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_description_versions_created_by_id_3f7e62a1 ON public.issue_description_versions USING btree (created_by_id);


--
-- Name: issue_description_versions_issue_id_c8baa13e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_description_versions_issue_id_c8baa13e ON public.issue_description_versions USING btree (issue_id);


--
-- Name: issue_description_versions_owned_by_id_0effe4d0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_description_versions_owned_by_id_0effe4d0 ON public.issue_description_versions USING btree (owned_by_id);


--
-- Name: issue_description_versions_project_id_536b23ef; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_description_versions_project_id_536b23ef ON public.issue_description_versions USING btree (project_id);


--
-- Name: issue_description_versions_updated_by_id_6530365d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_description_versions_updated_by_id_6530365d ON public.issue_description_versions USING btree (updated_by_id);


--
-- Name: issue_description_versions_workspace_id_88e930f9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_description_versions_workspace_id_88e930f9 ON public.issue_description_versions USING btree (workspace_id);


--
-- Name: issue_label_created_by_id_94075315; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_label_created_by_id_94075315 ON public.issue_labels USING btree (created_by_id);


--
-- Name: issue_label_issue_id_0f252e52; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_label_issue_id_0f252e52 ON public.issue_labels USING btree (issue_id);


--
-- Name: issue_label_label_id_5f22777f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_label_label_id_5f22777f ON public.issue_labels USING btree (label_id);


--
-- Name: issue_label_project_id_eaa2ba39; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_label_project_id_eaa2ba39 ON public.issue_labels USING btree (project_id);


--
-- Name: issue_label_updated_by_id_a97a6733; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_label_updated_by_id_a97a6733 ON public.issue_labels USING btree (updated_by_id);


--
-- Name: issue_label_workspace_id_b5b1faac; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_label_workspace_id_b5b1faac ON public.issue_labels USING btree (workspace_id);


--
-- Name: issue_links_created_by_id_5e4aa092; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_links_created_by_id_5e4aa092 ON public.issue_links USING btree (created_by_id);


--
-- Name: issue_links_issue_id_7032881f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_links_issue_id_7032881f ON public.issue_links USING btree (issue_id);


--
-- Name: issue_links_project_id_63d6e9ce; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_links_project_id_63d6e9ce ON public.issue_links USING btree (project_id);


--
-- Name: issue_links_updated_by_id_a771cce4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_links_updated_by_id_a771cce4 ON public.issue_links USING btree (updated_by_id);


--
-- Name: issue_links_workspace_id_ff9038e7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_links_workspace_id_ff9038e7 ON public.issue_links USING btree (workspace_id);


--
-- Name: issue_mention_unique_issue_mention_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_mention_unique_issue_mention_when_deleted_at_null ON public.issue_mentions USING btree (issue_id, mention_id) WHERE (deleted_at IS NULL);


--
-- Name: issue_mentions_created_by_id_eb44759e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_mentions_created_by_id_eb44759e ON public.issue_mentions USING btree (created_by_id);


--
-- Name: issue_mentions_issue_id_d8821107; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_mentions_issue_id_d8821107 ON public.issue_mentions USING btree (issue_id);


--
-- Name: issue_mentions_mention_id_cf1b9346; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_mentions_mention_id_cf1b9346 ON public.issue_mentions USING btree (mention_id);


--
-- Name: issue_mentions_project_id_d0cccdf5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_mentions_project_id_d0cccdf5 ON public.issue_mentions USING btree (project_id);


--
-- Name: issue_mentions_updated_by_id_c62106d3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_mentions_updated_by_id_c62106d3 ON public.issue_mentions USING btree (updated_by_id);


--
-- Name: issue_mentions_workspace_id_4ca59d05; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_mentions_workspace_id_4ca59d05 ON public.issue_mentions USING btree (workspace_id);


--
-- Name: issue_parent_id_ce8d76ba; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_parent_id_ce8d76ba ON public.issues USING btree (parent_id);


--
-- Name: issue_project_id_fea0fc80; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_project_id_fea0fc80 ON public.issues USING btree (project_id);


--
-- Name: issue_property_created_by_id_8e92131c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_property_created_by_id_8e92131c ON public.project_user_properties USING btree (created_by_id);


--
-- Name: issue_property_project_id_30e7de7b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_property_project_id_30e7de7b ON public.project_user_properties USING btree (project_id);


--
-- Name: issue_property_updated_by_id_ff158d4d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_property_updated_by_id_ff158d4d ON public.project_user_properties USING btree (updated_by_id);


--
-- Name: issue_property_user_id_0b1d1c8f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_property_user_id_0b1d1c8f ON public.project_user_properties USING btree (user_id);


--
-- Name: issue_property_workspace_id_17860d65; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_property_workspace_id_17860d65 ON public.project_user_properties USING btree (workspace_id);


--
-- Name: issue_reaction_unique_issue_actor_reaction_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_reaction_unique_issue_actor_reaction_when_deleted_at_null ON public.issue_reactions USING btree (issue_id, actor_id, reaction) WHERE (deleted_at IS NULL);


--
-- Name: issue_reactions_actor_id_5f5b8303; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_reactions_actor_id_5f5b8303 ON public.issue_reactions USING btree (actor_id);


--
-- Name: issue_reactions_created_by_id_3953b7de; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_reactions_created_by_id_3953b7de ON public.issue_reactions USING btree (created_by_id);


--
-- Name: issue_reactions_issue_id_2c324bae; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_reactions_issue_id_2c324bae ON public.issue_reactions USING btree (issue_id);


--
-- Name: issue_reactions_project_id_8708ecaf; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_reactions_project_id_8708ecaf ON public.issue_reactions USING btree (project_id);


--
-- Name: issue_reactions_updated_by_id_4069af90; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_reactions_updated_by_id_4069af90 ON public.issue_reactions USING btree (updated_by_id);


--
-- Name: issue_reactions_workspace_id_bd8d7550; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_reactions_workspace_id_bd8d7550 ON public.issue_reactions USING btree (workspace_id);


--
-- Name: issue_relation_unique_issue_related_issue_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_relation_unique_issue_related_issue_when_deleted_at_null ON public.issue_relations USING btree (issue_id, related_issue_id) WHERE (deleted_at IS NULL);


--
-- Name: issue_relations_created_by_id_854d07e7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_relations_created_by_id_854d07e7 ON public.issue_relations USING btree (created_by_id);


--
-- Name: issue_relations_issue_id_e1db6f72; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_relations_issue_id_e1db6f72 ON public.issue_relations USING btree (issue_id);


--
-- Name: issue_relations_project_id_15350161; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_relations_project_id_15350161 ON public.issue_relations USING btree (project_id);


--
-- Name: issue_relations_related_issue_id_e1ea44a7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_relations_related_issue_id_e1ea44a7 ON public.issue_relations USING btree (related_issue_id);


--
-- Name: issue_relations_updated_by_id_3dfa850f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_relations_updated_by_id_3dfa850f ON public.issue_relations USING btree (updated_by_id);


--
-- Name: issue_relations_workspace_id_00b50e90; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_relations_workspace_id_00b50e90 ON public.issue_relations USING btree (workspace_id);


--
-- Name: issue_sequence_created_by_id_59270506; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_sequence_created_by_id_59270506 ON public.issue_sequences USING btree (created_by_id);


--
-- Name: issue_sequence_issue_id_16e9f00f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_sequence_issue_id_16e9f00f ON public.issue_sequences USING btree (issue_id);


--
-- Name: issue_sequence_project_id_ce882e85; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_sequence_project_id_ce882e85 ON public.issue_sequences USING btree (project_id);


--
-- Name: issue_sequence_updated_by_id_310c8dd3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_sequence_updated_by_id_310c8dd3 ON public.issue_sequences USING btree (updated_by_id);


--
-- Name: issue_sequence_workspace_id_0d3f0fd4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_sequence_workspace_id_0d3f0fd4 ON public.issue_sequences USING btree (workspace_id);


--
-- Name: issue_sequences_sequence_2c9458d4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_sequences_sequence_2c9458d4 ON public.issue_sequences USING btree (sequence);


--
-- Name: issue_state_id_1a65560d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_state_id_1a65560d ON public.issues USING btree (state_id);


--
-- Name: issue_subscriber_unique_issue_subscriber_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_subscriber_unique_issue_subscriber_when_deleted_at_null ON public.issue_subscribers USING btree (issue_id, subscriber_id) WHERE (deleted_at IS NULL);


--
-- Name: issue_subscribers_created_by_id_b6ea0157; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_subscribers_created_by_id_b6ea0157 ON public.issue_subscribers USING btree (created_by_id);


--
-- Name: issue_subscribers_issue_id_85cf2093; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_subscribers_issue_id_85cf2093 ON public.issue_subscribers USING btree (issue_id);


--
-- Name: issue_subscribers_project_id_cf48d75f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_subscribers_project_id_cf48d75f ON public.issue_subscribers USING btree (project_id);


--
-- Name: issue_subscribers_subscriber_id_2d89c988; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_subscribers_subscriber_id_2d89c988 ON public.issue_subscribers USING btree (subscriber_id);


--
-- Name: issue_subscribers_updated_by_id_1bfc2f55; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_subscribers_updated_by_id_1bfc2f55 ON public.issue_subscribers USING btree (updated_by_id);


--
-- Name: issue_subscribers_workspace_id_96afa91f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_subscribers_workspace_id_96afa91f ON public.issue_subscribers USING btree (workspace_id);


--
-- Name: issue_types_created_by_id_48764f53; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_types_created_by_id_48764f53 ON public.issue_types USING btree (created_by_id);


--
-- Name: issue_types_updated_by_id_4919203b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_types_updated_by_id_4919203b ON public.issue_types USING btree (updated_by_id);


--
-- Name: issue_types_workspace_id_591c6f3b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_types_workspace_id_591c6f3b ON public.issue_types USING btree (workspace_id);


--
-- Name: issue_updated_by_id_f1261863; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_updated_by_id_f1261863 ON public.issues USING btree (updated_by_id);


--
-- Name: issue_versions_activity_id_b1872ffc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_activity_id_b1872ffc ON public.issue_versions USING btree (activity_id);


--
-- Name: issue_versions_created_by_id_a782830a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_created_by_id_a782830a ON public.issue_versions USING btree (created_by_id);


--
-- Name: issue_versions_issue_id_25cf001c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_issue_id_25cf001c ON public.issue_versions USING btree (issue_id);


--
-- Name: issue_versions_owned_by_id_7586378d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_owned_by_id_7586378d ON public.issue_versions USING btree (owned_by_id);


--
-- Name: issue_versions_project_id_a069ad03; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_project_id_a069ad03 ON public.issue_versions USING btree (project_id);


--
-- Name: issue_versions_updated_by_id_dcae6dd2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_updated_by_id_dcae6dd2 ON public.issue_versions USING btree (updated_by_id);


--
-- Name: issue_versions_workspace_id_b8c48b7c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_versions_workspace_id_b8c48b7c ON public.issue_versions USING btree (workspace_id);


--
-- Name: issue_views_created_by_id_0d2e456b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_views_created_by_id_0d2e456b ON public.issue_views USING btree (created_by_id);


--
-- Name: issue_views_owned_by_id_5e261e5d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_views_owned_by_id_5e261e5d ON public.issue_views USING btree (owned_by_id);


--
-- Name: issue_views_project_id_55ee009f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_views_project_id_55ee009f ON public.issue_views USING btree (project_id);


--
-- Name: issue_views_updated_by_id_28cd9870; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_views_updated_by_id_28cd9870 ON public.issue_views USING btree (updated_by_id);


--
-- Name: issue_views_workspace_id_8785e03d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_views_workspace_id_8785e03d ON public.issue_views USING btree (workspace_id);


--
-- Name: issue_vote_unique_issue_actor_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_vote_unique_issue_actor_when_deleted_at_null ON public.issue_votes USING btree (issue_id, actor_id) WHERE (deleted_at IS NULL);


--
-- Name: issue_votes_actor_id_525cab61; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_votes_actor_id_525cab61 ON public.issue_votes USING btree (actor_id);


--
-- Name: issue_votes_created_by_id_86adcf5c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_votes_created_by_id_86adcf5c ON public.issue_votes USING btree (created_by_id);


--
-- Name: issue_votes_issue_id_07a61ecb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_votes_issue_id_07a61ecb ON public.issue_votes USING btree (issue_id);


--
-- Name: issue_votes_project_id_b649f55b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_votes_project_id_b649f55b ON public.issue_votes USING btree (project_id);


--
-- Name: issue_votes_updated_by_id_9e2a6cdc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_votes_updated_by_id_9e2a6cdc ON public.issue_votes USING btree (updated_by_id);


--
-- Name: issue_votes_workspace_id_a3e91a6b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_votes_workspace_id_a3e91a6b ON public.issue_votes USING btree (workspace_id);


--
-- Name: issue_workspace_id_c84878c1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issue_workspace_id_c84878c1 ON public.issues USING btree (workspace_id);


--
-- Name: issues_estimate_point_id_a6822abe; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issues_estimate_point_id_a6822abe ON public.issues USING btree (estimate_point_id);


--
-- Name: issues_type_id_a4710b19; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX issues_type_id_a4710b19 ON public.issues USING btree (type_id);


--
-- Name: label_created_by_id_aa6ffcfa; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX label_created_by_id_aa6ffcfa ON public.labels USING btree (created_by_id);


--
-- Name: label_parent_id_7a853296; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX label_parent_id_7a853296 ON public.labels USING btree (parent_id);


--
-- Name: label_project_id_90e0f1a2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX label_project_id_90e0f1a2 ON public.labels USING btree (project_id);


--
-- Name: label_updated_by_id_894a5464; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX label_updated_by_id_894a5464 ON public.labels USING btree (updated_by_id);


--
-- Name: label_workspace_id_c4c9ae5a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX label_workspace_id_c4c9ae5a ON public.labels USING btree (workspace_id);


--
-- Name: module_created_by_id_ff7a5866; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_created_by_id_ff7a5866 ON public.modules USING btree (created_by_id);


--
-- Name: module_draft_issue_unique_issue_module_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX module_draft_issue_unique_issue_module_when_deleted_at_null ON public.draft_issue_modules USING btree (draft_issue_id, module_id) WHERE (deleted_at IS NULL);


--
-- Name: module_issue_unique_issue_module_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX module_issue_unique_issue_module_when_deleted_at_null ON public.module_issues USING btree (issue_id, module_id) WHERE (deleted_at IS NULL);


--
-- Name: module_issues_created_by_id_de0b995a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_issues_created_by_id_de0b995a ON public.module_issues USING btree (created_by_id);


--
-- Name: module_issues_issue_id_7caa908b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_issues_issue_id_7caa908b ON public.module_issues USING btree (issue_id);


--
-- Name: module_issues_module_id_74e0ed5a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_issues_module_id_74e0ed5a ON public.module_issues USING btree (module_id);


--
-- Name: module_issues_project_id_59836d1e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_issues_project_id_59836d1e ON public.module_issues USING btree (project_id);


--
-- Name: module_issues_updated_by_id_46dbf724; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_issues_updated_by_id_46dbf724 ON public.module_issues USING btree (updated_by_id);


--
-- Name: module_issues_workspace_id_6bf85201; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_issues_workspace_id_6bf85201 ON public.module_issues USING btree (workspace_id);


--
-- Name: module_lead_id_04966630; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_lead_id_04966630 ON public.modules USING btree (lead_id);


--
-- Name: module_links_created_by_id_eaf6492f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_links_created_by_id_eaf6492f ON public.module_links USING btree (created_by_id);


--
-- Name: module_links_module_id_0fda3f8a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_links_module_id_0fda3f8a ON public.module_links USING btree (module_id);


--
-- Name: module_links_project_id_f720bb79; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_links_project_id_f720bb79 ON public.module_links USING btree (project_id);


--
-- Name: module_links_updated_by_id_4da419e7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_links_updated_by_id_4da419e7 ON public.module_links USING btree (updated_by_id);


--
-- Name: module_links_workspace_id_0521c11c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_links_workspace_id_0521c11c ON public.module_links USING btree (workspace_id);


--
-- Name: module_member_created_by_id_2ed84a65; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_member_created_by_id_2ed84a65 ON public.module_members USING btree (created_by_id);


--
-- Name: module_member_member_id_928f473e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_member_member_id_928f473e ON public.module_members USING btree (member_id);


--
-- Name: module_member_module_id_f00be7ef; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_member_module_id_f00be7ef ON public.module_members USING btree (module_id);


--
-- Name: module_member_project_id_ec8d2376; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_member_project_id_ec8d2376 ON public.module_members USING btree (project_id);


--
-- Name: module_member_unique_module_member_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX module_member_unique_module_member_when_deleted_at_null ON public.module_members USING btree (module_id, member_id) WHERE (deleted_at IS NULL);


--
-- Name: module_member_updated_by_id_a9046438; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_member_updated_by_id_a9046438 ON public.module_members USING btree (updated_by_id);


--
-- Name: module_member_workspace_id_f2f23c73; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_member_workspace_id_f2f23c73 ON public.module_members USING btree (workspace_id);


--
-- Name: module_project_id_da84b04f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_project_id_da84b04f ON public.modules USING btree (project_id);


--
-- Name: module_unique_name_project_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX module_unique_name_project_when_deleted_at_null ON public.modules USING btree (name, project_id) WHERE (deleted_at IS NULL);


--
-- Name: module_updated_by_id_72ab6d5c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_updated_by_id_72ab6d5c ON public.modules USING btree (updated_by_id);


--
-- Name: module_user_properties_created_by_id_bdd98440; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_user_properties_created_by_id_bdd98440 ON public.module_user_properties USING btree (created_by_id);


--
-- Name: module_user_properties_module_id_e95b158a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_user_properties_module_id_e95b158a ON public.module_user_properties USING btree (module_id);


--
-- Name: module_user_properties_project_id_3c5a4972; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_user_properties_project_id_3c5a4972 ON public.module_user_properties USING btree (project_id);


--
-- Name: module_user_properties_unique_module_user_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX module_user_properties_unique_module_user_when_deleted_at_null ON public.module_user_properties USING btree (module_id, user_id) WHERE (deleted_at IS NULL);


--
-- Name: module_user_properties_updated_by_id_b7dafc77; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_user_properties_updated_by_id_b7dafc77 ON public.module_user_properties USING btree (updated_by_id);


--
-- Name: module_user_properties_user_id_e83a1c2c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_user_properties_user_id_e83a1c2c ON public.module_user_properties USING btree (user_id);


--
-- Name: module_user_properties_workspace_id_ddaf807c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_user_properties_workspace_id_ddaf807c ON public.module_user_properties USING btree (workspace_id);


--
-- Name: module_workspace_id_0a826fef; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX module_workspace_id_0a826fef ON public.modules USING btree (workspace_id);


--
-- Name: notif_entity_identifier_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_entity_identifier_idx ON public.notifications USING btree (entity_identifier);


--
-- Name: notif_entity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_entity_idx ON public.notifications USING btree (receiver_id, read_at);


--
-- Name: notif_entity_lookup_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_entity_lookup_idx ON public.notifications USING btree (workspace_id, entity_identifier, entity_name);


--
-- Name: notif_entity_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_entity_name_idx ON public.notifications USING btree (entity_name);


--
-- Name: notif_read_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_read_at_idx ON public.notifications USING btree (read_at);


--
-- Name: notif_receiver_entity_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_receiver_entity_idx ON public.notifications USING btree (receiver_id, workspace_id, entity_name, read_at);


--
-- Name: notif_receiver_sender_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_receiver_sender_idx ON public.notifications USING btree (receiver_id, workspace_id, sender);


--
-- Name: notif_receiver_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_receiver_state_idx ON public.notifications USING btree (receiver_id, workspace_id, snoozed_till, archived_at);


--
-- Name: notif_receiver_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notif_receiver_status_idx ON public.notifications USING btree (receiver_id, workspace_id, read_at, created_at);


--
-- Name: notifications_created_by_id_b9c3f81b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_created_by_id_b9c3f81b ON public.notifications USING btree (created_by_id);


--
-- Name: notifications_project_id_e4d4f192; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_project_id_e4d4f192 ON public.notifications USING btree (project_id);


--
-- Name: notifications_receiver_id_b708b2b0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_receiver_id_b708b2b0 ON public.notifications USING btree (receiver_id);


--
-- Name: notifications_triggered_by_id_31cdec21; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_triggered_by_id_31cdec21 ON public.notifications USING btree (triggered_by_id);


--
-- Name: notifications_updated_by_id_8a651e96; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_updated_by_id_8a651e96 ON public.notifications USING btree (updated_by_id);


--
-- Name: notifications_workspace_id_b2f09ef7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX notifications_workspace_id_b2f09ef7 ON public.notifications USING btree (workspace_id);


--
-- Name: page_labels_created_by_id_fbd942c0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_labels_created_by_id_fbd942c0 ON public.page_labels USING btree (created_by_id);


--
-- Name: page_labels_label_id_05958e53; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_labels_label_id_05958e53 ON public.page_labels USING btree (label_id);


--
-- Name: page_labels_page_id_0e6cdb3d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_labels_page_id_0e6cdb3d ON public.page_labels USING btree (page_id);


--
-- Name: page_labels_updated_by_id_d9fddbff; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_labels_updated_by_id_d9fddbff ON public.page_labels USING btree (updated_by_id);


--
-- Name: page_labels_workspace_id_078bb01c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_labels_workspace_id_078bb01c ON public.page_labels USING btree (workspace_id);


--
-- Name: page_logs_created_by_id_4a295aec; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_logs_created_by_id_4a295aec ON public.page_logs USING btree (created_by_id);


--
-- Name: page_logs_page_id_0e0d747d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_logs_page_id_0e0d747d ON public.page_logs USING btree (page_id);


--
-- Name: page_logs_updated_by_id_1995190b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_logs_updated_by_id_1995190b ON public.page_logs USING btree (updated_by_id);


--
-- Name: page_logs_workspace_id_be7bde64; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_logs_workspace_id_be7bde64 ON public.page_logs USING btree (workspace_id);


--
-- Name: page_versions_created_by_id_d660b13b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_versions_created_by_id_d660b13b ON public.page_versions USING btree (created_by_id);


--
-- Name: page_versions_owned_by_id_6d9143db; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_versions_owned_by_id_6d9143db ON public.page_versions USING btree (owned_by_id);


--
-- Name: page_versions_page_id_c46471da; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_versions_page_id_c46471da ON public.page_versions USING btree (page_id);


--
-- Name: page_versions_updated_by_id_72d5e579; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_versions_updated_by_id_72d5e579 ON public.page_versions USING btree (updated_by_id);


--
-- Name: page_versions_workspace_id_8330a200; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX page_versions_workspace_id_8330a200 ON public.page_versions USING btree (workspace_id);


--
-- Name: pagelog_entity_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pagelog_entity_id_idx ON public.page_logs USING btree (entity_identifier);


--
-- Name: pagelog_entity_name_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pagelog_entity_name_idx ON public.page_logs USING btree (entity_name);


--
-- Name: pagelog_entity_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pagelog_entity_type_idx ON public.page_logs USING btree (entity_type);


--
-- Name: pagelog_name_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pagelog_name_id_idx ON public.page_logs USING btree (entity_name, entity_identifier);


--
-- Name: pagelog_type_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pagelog_type_id_idx ON public.page_logs USING btree (entity_type, entity_identifier);


--
-- Name: pages_created_by_id_d109a675; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pages_created_by_id_d109a675 ON public.pages USING btree (created_by_id);


--
-- Name: pages_owned_by_id_bf50485f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pages_owned_by_id_bf50485f ON public.pages USING btree (owned_by_id);


--
-- Name: pages_parent_id_8b823409; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pages_parent_id_8b823409 ON public.pages USING btree (parent_id);


--
-- Name: pages_updated_by_id_6c42de3e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pages_updated_by_id_6c42de3e ON public.pages USING btree (updated_by_id);


--
-- Name: pages_workspace_id_c6c51010; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX pages_workspace_id_c6c51010 ON public.pages USING btree (workspace_id);


--
-- Name: project_created_by_id_6cc13408; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_created_by_id_6cc13408 ON public.projects USING btree (created_by_id);


--
-- Name: project_default_assignee_id_6ba45f90; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_default_assignee_id_6ba45f90 ON public.projects USING btree (default_assignee_id);


--
-- Name: project_deploy_boards_anchor_b61b8817_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_deploy_boards_anchor_b61b8817_like ON public.project_deploy_boards USING btree (anchor varchar_pattern_ops);


--
-- Name: project_deploy_boards_created_by_id_2ea72f98; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_deploy_boards_created_by_id_2ea72f98 ON public.project_deploy_boards USING btree (created_by_id);


--
-- Name: project_deploy_boards_inbox_id_a6a75525; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_deploy_boards_inbox_id_a6a75525 ON public.project_deploy_boards USING btree (intake_id);


--
-- Name: project_deploy_boards_project_id_49d887b2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_deploy_boards_project_id_49d887b2 ON public.project_deploy_boards USING btree (project_id);


--
-- Name: project_deploy_boards_updated_by_id_290eb99e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_deploy_boards_updated_by_id_290eb99e ON public.project_deploy_boards USING btree (updated_by_id);


--
-- Name: project_deploy_boards_workspace_id_cd92f164; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_deploy_boards_workspace_id_cd92f164 ON public.project_deploy_boards USING btree (workspace_id);


--
-- Name: project_identifier_created_by_id_2b6f273a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_identifier_created_by_id_2b6f273a ON public.project_identifiers USING btree (created_by_id);


--
-- Name: project_identifier_updated_by_id_1a00e2a0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_identifier_updated_by_id_1a00e2a0 ON public.project_identifiers USING btree (updated_by_id);


--
-- Name: project_identifier_workspace_id_6024b517; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_identifier_workspace_id_6024b517 ON public.project_identifiers USING btree (workspace_id);


--
-- Name: project_identifiers_name_6ca8a4b0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_identifiers_name_6ca8a4b0 ON public.project_identifiers USING btree (name);


--
-- Name: project_identifiers_name_6ca8a4b0_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_identifiers_name_6ca8a4b0_like ON public.project_identifiers USING btree (name varchar_pattern_ops);


--
-- Name: project_issue_type_unique_project_issue_type_when_deleted_at_nu; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_issue_type_unique_project_issue_type_when_deleted_at_nu ON public.project_issue_types USING btree (project_id, issue_type_id) WHERE (deleted_at IS NULL);


--
-- Name: project_issue_types_created_by_id_049cecfd; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_issue_types_created_by_id_049cecfd ON public.project_issue_types USING btree (created_by_id);


--
-- Name: project_issue_types_issue_type_id_9494de9f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_issue_types_issue_type_id_9494de9f ON public.project_issue_types USING btree (issue_type_id);


--
-- Name: project_issue_types_project_id_ef6e52e4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_issue_types_project_id_ef6e52e4 ON public.project_issue_types USING btree (project_id);


--
-- Name: project_issue_types_updated_by_id_b5998397; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_issue_types_updated_by_id_b5998397 ON public.project_issue_types USING btree (updated_by_id);


--
-- Name: project_issue_types_workspace_id_ace3c5b5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_issue_types_workspace_id_ace3c5b5 ON public.project_issue_types USING btree (workspace_id);


--
-- Name: project_member_created_by_id_8b363306; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_created_by_id_8b363306 ON public.project_members USING btree (created_by_id);


--
-- Name: project_member_invite_created_by_id_a87df45c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_invite_created_by_id_a87df45c ON public.project_member_invites USING btree (created_by_id);


--
-- Name: project_member_invite_project_id_8fb7750e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_invite_project_id_8fb7750e ON public.project_member_invites USING btree (project_id);


--
-- Name: project_member_invite_updated_by_id_5aa55c96; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_invite_updated_by_id_5aa55c96 ON public.project_member_invites USING btree (updated_by_id);


--
-- Name: project_member_invite_workspace_id_64e2dc4c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_invite_workspace_id_64e2dc4c ON public.project_member_invites USING btree (workspace_id);


--
-- Name: project_member_member_id_9d6b126b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_member_id_9d6b126b ON public.project_members USING btree (member_id);


--
-- Name: project_member_project_id_11ea1a9e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_project_id_11ea1a9e ON public.project_members USING btree (project_id);


--
-- Name: project_member_unique_project_member_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_member_unique_project_member_when_deleted_at_null ON public.project_members USING btree (project_id, member_id) WHERE (deleted_at IS NULL);


--
-- Name: project_member_updated_by_id_cf6aaac4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_updated_by_id_cf6aaac4 ON public.project_members USING btree (updated_by_id);


--
-- Name: project_member_workspace_id_88bb9a97; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_member_workspace_id_88bb9a97 ON public.project_members USING btree (workspace_id);


--
-- Name: project_page_unique_project_page_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_page_unique_project_page_when_deleted_at_null ON public.project_pages USING btree (project_id, page_id) WHERE (deleted_at IS NULL);


--
-- Name: project_pages_created_by_id_b9d02062; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_pages_created_by_id_b9d02062 ON public.project_pages USING btree (created_by_id);


--
-- Name: project_pages_page_id_a0f54439; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_pages_page_id_a0f54439 ON public.project_pages USING btree (page_id);


--
-- Name: project_pages_project_id_376ba35a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_pages_project_id_376ba35a ON public.project_pages USING btree (project_id);


--
-- Name: project_pages_updated_by_id_b80bf0f4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_pages_updated_by_id_b80bf0f4 ON public.project_pages USING btree (updated_by_id);


--
-- Name: project_pages_workspace_id_13ed9e73; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_pages_workspace_id_13ed9e73 ON public.project_pages USING btree (workspace_id);


--
-- Name: project_project_lead_id_caf8e353; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_project_lead_id_caf8e353 ON public.projects USING btree (project_lead_id);


--
-- Name: project_public_member_unique_project_member_when_deleted_at_nul; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_public_member_unique_project_member_when_deleted_at_nul ON public.project_public_members USING btree (project_id, member_id) WHERE (deleted_at IS NULL);


--
-- Name: project_public_members_created_by_id_c4c7c776; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_public_members_created_by_id_c4c7c776 ON public.project_public_members USING btree (created_by_id);


--
-- Name: project_public_members_member_id_52f257f9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_public_members_member_id_52f257f9 ON public.project_public_members USING btree (member_id);


--
-- Name: project_public_members_project_id_2dfd893d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_public_members_project_id_2dfd893d ON public.project_public_members USING btree (project_id);


--
-- Name: project_public_members_updated_by_id_c3e4d675; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_public_members_updated_by_id_c3e4d675 ON public.project_public_members USING btree (updated_by_id);


--
-- Name: project_public_members_workspace_id_ebfce110; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_public_members_workspace_id_ebfce110 ON public.project_public_members USING btree (workspace_id);


--
-- Name: project_unique_identifier_workspace_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_unique_identifier_workspace_when_deleted_at_null ON public.projects USING btree (identifier, workspace_id) WHERE (deleted_at IS NULL);


--
-- Name: project_unique_name_workspace_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_unique_name_workspace_when_deleted_at_null ON public.projects USING btree (name, workspace_id) WHERE (deleted_at IS NULL);


--
-- Name: project_updated_by_id_fe290525; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_updated_by_id_fe290525 ON public.projects USING btree (updated_by_id);


--
-- Name: project_user_property_unique_user_project_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_user_property_unique_user_project_when_deleted_at_null ON public.project_user_properties USING btree (user_id, project_id) WHERE (deleted_at IS NULL);


--
-- Name: project_webhook_unique_project_webhook_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX project_webhook_unique_project_webhook_when_deleted_at_null ON public.project_webhooks USING btree (project_id, webhook_id) WHERE (deleted_at IS NULL);


--
-- Name: project_webhooks_created_by_id_c3e4bfa3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_webhooks_created_by_id_c3e4bfa3 ON public.project_webhooks USING btree (created_by_id);


--
-- Name: project_webhooks_project_id_bec3cf8c; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_webhooks_project_id_bec3cf8c ON public.project_webhooks USING btree (project_id);


--
-- Name: project_webhooks_updated_by_id_a0183aeb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_webhooks_updated_by_id_a0183aeb ON public.project_webhooks USING btree (updated_by_id);


--
-- Name: project_webhooks_webhook_id_da27c6a7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_webhooks_webhook_id_da27c6a7 ON public.project_webhooks USING btree (webhook_id);


--
-- Name: project_webhooks_workspace_id_429ebf05; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_webhooks_workspace_id_429ebf05 ON public.project_webhooks USING btree (workspace_id);


--
-- Name: project_workspace_id_01764ff9; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX project_workspace_id_01764ff9 ON public.projects USING btree (workspace_id);


--
-- Name: projects_cover_image_asset_id_e6636b92; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX projects_cover_image_asset_id_e6636b92 ON public.projects USING btree (cover_image_asset_id);


--
-- Name: projects_default_state_id_f13e8b95; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX projects_default_state_id_f13e8b95 ON public.projects USING btree (default_state_id);


--
-- Name: projects_estimate_id_85c7b2ac; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX projects_estimate_id_85c7b2ac ON public.projects USING btree (estimate_id);


--
-- Name: projects_identifier_3267ade8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX projects_identifier_3267ade8 ON public.projects USING btree (identifier);


--
-- Name: projects_identifier_3267ade8_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX projects_identifier_3267ade8_like ON public.projects USING btree (identifier varchar_pattern_ops);


--
-- Name: sessions_expire_date_16e4c444; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_expire_date_16e4c444 ON public.sessions USING btree (expire_date);


--
-- Name: sessions_session_key_58f9471b_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_session_key_58f9471b_like ON public.sessions USING btree (session_key varchar_pattern_ops);


--
-- Name: sessions_user_id_05e26f4a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_user_id_05e26f4a ON public.sessions USING btree (user_id);


--
-- Name: sessions_user_id_05e26f4a_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sessions_user_id_05e26f4a_like ON public.sessions USING btree (user_id varchar_pattern_ops);


--
-- Name: slack_project_syncs_created_by_id_ec405a17; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX slack_project_syncs_created_by_id_ec405a17 ON public.slack_project_syncs USING btree (created_by_id);


--
-- Name: slack_project_syncs_project_id_016dc792; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX slack_project_syncs_project_id_016dc792 ON public.slack_project_syncs USING btree (project_id);


--
-- Name: slack_project_syncs_updated_by_id_152eb3b5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX slack_project_syncs_updated_by_id_152eb3b5 ON public.slack_project_syncs USING btree (updated_by_id);


--
-- Name: slack_project_syncs_workspace_id_d1822b06; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX slack_project_syncs_workspace_id_d1822b06 ON public.slack_project_syncs USING btree (workspace_id);


--
-- Name: slack_project_syncs_workspace_integration_id_d89c9b40; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX slack_project_syncs_workspace_integration_id_d89c9b40 ON public.slack_project_syncs USING btree (workspace_integration_id);


--
-- Name: social_login_connection_created_by_id_7ca2ef50; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX social_login_connection_created_by_id_7ca2ef50 ON public.social_login_connections USING btree (created_by_id);


--
-- Name: social_login_connection_updated_by_id_c13deb42; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX social_login_connection_updated_by_id_c13deb42 ON public.social_login_connections USING btree (updated_by_id);


--
-- Name: social_login_connection_user_id_0e26c0c5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX social_login_connection_user_id_0e26c0c5 ON public.social_login_connections USING btree (user_id);


--
-- Name: state_created_by_id_ff51a50d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX state_created_by_id_ff51a50d ON public.states USING btree (created_by_id);


--
-- Name: state_project_id_23a65fd6; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX state_project_id_23a65fd6 ON public.states USING btree (project_id);


--
-- Name: state_slug_bab0af35; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX state_slug_bab0af35 ON public.states USING btree (slug);


--
-- Name: state_slug_bab0af35_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX state_slug_bab0af35_like ON public.states USING btree (slug varchar_pattern_ops);


--
-- Name: state_unique_name_project_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX state_unique_name_project_when_deleted_at_null ON public.states USING btree (name, project_id) WHERE (deleted_at IS NULL);


--
-- Name: state_updated_by_id_be298453; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX state_updated_by_id_be298453 ON public.states USING btree (updated_by_id);


--
-- Name: state_workspace_id_2293282d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX state_workspace_id_2293282d ON public.states USING btree (workspace_id);


--
-- Name: stickies_created_by_id_f72e05c4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stickies_created_by_id_f72e05c4 ON public.stickies USING btree (created_by_id);


--
-- Name: stickies_owner_id_6ee3be2b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stickies_owner_id_6ee3be2b ON public.stickies USING btree (owner_id);


--
-- Name: stickies_updated_by_id_d660f1fb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stickies_updated_by_id_d660f1fb ON public.stickies USING btree (updated_by_id);


--
-- Name: stickies_workspace_id_0094496a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX stickies_workspace_id_0094496a ON public.stickies USING btree (workspace_id);


--
-- Name: team_created_by_id_725a9101; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX team_created_by_id_725a9101 ON public.teams USING btree (created_by_id);


--
-- Name: team_unique_name_workspace_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX team_unique_name_workspace_when_deleted_at_null ON public.teams USING btree (name, workspace_id) WHERE (deleted_at IS NULL);


--
-- Name: team_updated_by_id_79bb36f2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX team_updated_by_id_79bb36f2 ON public.teams USING btree (updated_by_id);


--
-- Name: team_workspace_id_1d56407f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX team_workspace_id_1d56407f ON public.teams USING btree (workspace_id);


--
-- Name: unique_name_when_project_null_and_not_deleted; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_name_when_project_null_and_not_deleted ON public.labels USING btree (name) WHERE ((deleted_at IS NULL) AND (project_id IS NULL));


--
-- Name: unique_name_workspace_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_name_workspace_when_deleted_at_null ON public.project_identifiers USING btree (name, workspace_id) WHERE (deleted_at IS NULL);


--
-- Name: unique_project_name_when_not_deleted; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_project_name_when_not_deleted ON public.labels USING btree (project_id, name) WHERE ((deleted_at IS NULL) AND (project_id IS NOT NULL));


--
-- Name: user_email_54dc62b2_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_email_54dc62b2_like ON public.users USING btree (email varchar_pattern_ops);


--
-- Name: user_favorite_unique_entity_type_entity_identifier_user_when_de; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX user_favorite_unique_entity_type_entity_identifier_user_when_de ON public.user_favorites USING btree (entity_type, entity_identifier, user_id) WHERE (deleted_at IS NULL);


--
-- Name: user_favorites_created_by_id_dc025309; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_favorites_created_by_id_dc025309 ON public.user_favorites USING btree (created_by_id);


--
-- Name: user_favorites_parent_id_550512e4; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_favorites_parent_id_550512e4 ON public.user_favorites USING btree (parent_id);


--
-- Name: user_favorites_project_id_359b527f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_favorites_project_id_359b527f ON public.user_favorites USING btree (project_id);


--
-- Name: user_favorites_updated_by_id_a1a5ac4a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_favorites_updated_by_id_a1a5ac4a ON public.user_favorites USING btree (updated_by_id);


--
-- Name: user_favorites_user_id_cea7e2d2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_favorites_user_id_cea7e2d2 ON public.user_favorites USING btree (user_id);


--
-- Name: user_favorites_workspace_id_aa90f680; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_favorites_workspace_id_aa90f680 ON public.user_favorites USING btree (workspace_id);


--
-- Name: user_notification_preferences_created_by_id_54dc743a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_notification_preferences_created_by_id_54dc743a ON public.user_notification_preferences USING btree (created_by_id);


--
-- Name: user_notification_preferences_project_id_e0ca17f8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_notification_preferences_project_id_e0ca17f8 ON public.user_notification_preferences USING btree (project_id);


--
-- Name: user_notification_preferences_updated_by_id_eb70a86d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_notification_preferences_updated_by_id_eb70a86d ON public.user_notification_preferences USING btree (updated_by_id);


--
-- Name: user_notification_preferences_user_id_9dccc056; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_notification_preferences_user_id_9dccc056 ON public.user_notification_preferences USING btree (user_id);


--
-- Name: user_notification_preferences_workspace_id_a2321c58; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_notification_preferences_workspace_id_a2321c58 ON public.user_notification_preferences USING btree (workspace_id);


--
-- Name: user_recent_visits_created_by_id_a655b75f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_recent_visits_created_by_id_a655b75f ON public.user_recent_visits USING btree (created_by_id);


--
-- Name: user_recent_visits_project_id_e5eecf27; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_recent_visits_project_id_e5eecf27 ON public.user_recent_visits USING btree (project_id);


--
-- Name: user_recent_visits_updated_by_id_42b12ef2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_recent_visits_updated_by_id_42b12ef2 ON public.user_recent_visits USING btree (updated_by_id);


--
-- Name: user_recent_visits_user_id_f5153288; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_recent_visits_user_id_f5153288 ON public.user_recent_visits USING btree (user_id);


--
-- Name: user_recent_visits_workspace_id_362a4e80; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_recent_visits_workspace_id_362a4e80 ON public.user_recent_visits USING btree (workspace_id);


--
-- Name: user_username_cf016618_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_username_cf016618_like ON public.users USING btree (username varchar_pattern_ops);


--
-- Name: users_avatar_asset_id_50fa2043; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_avatar_asset_id_50fa2043 ON public.users USING btree (avatar_asset_id);


--
-- Name: users_cover_image_asset_id_b9679cbc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_cover_image_asset_id_b9679cbc ON public.users USING btree (cover_image_asset_id);


--
-- Name: webhook_logs_created_by_id_71e7bc38; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_created_by_id_71e7bc38 ON public.webhook_logs USING btree (created_by_id);


--
-- Name: webhook_logs_updated_by_id_3d9bad04; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_updated_by_id_3d9bad04 ON public.webhook_logs USING btree (updated_by_id);


--
-- Name: webhook_logs_workspace_id_ffcd0e31; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhook_logs_workspace_id_ffcd0e31 ON public.webhook_logs USING btree (workspace_id);


--
-- Name: webhook_url_unique_url_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX webhook_url_unique_url_when_deleted_at_null ON public.webhooks USING btree (workspace_id, url) WHERE (deleted_at IS NULL);


--
-- Name: webhooks_created_by_id_25aca1b0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhooks_created_by_id_25aca1b0 ON public.webhooks USING btree (created_by_id);


--
-- Name: webhooks_updated_by_id_ea35154e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhooks_updated_by_id_ea35154e ON public.webhooks USING btree (updated_by_id);


--
-- Name: webhooks_workspace_id_da5865d7; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX webhooks_workspace_id_da5865d7 ON public.webhooks USING btree (workspace_id);


--
-- Name: workspace_created_by_id_10ad894e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_created_by_id_10ad894e ON public.workspaces USING btree (created_by_id);


--
-- Name: workspace_home_preferences_created_by_id_f31fc163; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_home_preferences_created_by_id_f31fc163 ON public.workspace_home_preferences USING btree (created_by_id);


--
-- Name: workspace_home_preferences_updated_by_id_14ed118a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_home_preferences_updated_by_id_14ed118a ON public.workspace_home_preferences USING btree (updated_by_id);


--
-- Name: workspace_home_preferences_user_id_4087938d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_home_preferences_user_id_4087938d ON public.workspace_home_preferences USING btree (user_id);


--
-- Name: workspace_home_preferences_workspace_id_b49f76e0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_home_preferences_workspace_id_b49f76e0 ON public.workspace_home_preferences USING btree (workspace_id);


--
-- Name: workspace_integrations_actor_id_21619aa1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_integrations_actor_id_21619aa1 ON public.workspace_integrations USING btree (actor_id);


--
-- Name: workspace_integrations_api_token_id_bdb1759b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_integrations_api_token_id_bdb1759b ON public.workspace_integrations USING btree (api_token_id);


--
-- Name: workspace_integrations_created_by_id_37639c73; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_integrations_created_by_id_37639c73 ON public.workspace_integrations USING btree (created_by_id);


--
-- Name: workspace_integrations_integration_id_6cb0aace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_integrations_integration_id_6cb0aace ON public.workspace_integrations USING btree (integration_id);


--
-- Name: workspace_integrations_updated_by_id_fce01dcb; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_integrations_updated_by_id_fce01dcb ON public.workspace_integrations USING btree (updated_by_id);


--
-- Name: workspace_integrations_workspace_id_27ebeb6b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_integrations_workspace_id_27ebeb6b ON public.workspace_integrations USING btree (workspace_id);


--
-- Name: workspace_member_created_by_id_8dc8b040; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_created_by_id_8dc8b040 ON public.workspace_members USING btree (created_by_id);


--
-- Name: workspace_member_invite_created_by_id_082f21d3; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_invite_created_by_id_082f21d3 ON public.workspace_member_invites USING btree (created_by_id);


--
-- Name: workspace_member_invite_unique_email_workspace_when_deleted_at_; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX workspace_member_invite_unique_email_workspace_when_deleted_at_ ON public.workspace_member_invites USING btree (email, workspace_id) WHERE (deleted_at IS NULL);


--
-- Name: workspace_member_invite_updated_by_id_d31a9c7f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_invite_updated_by_id_d31a9c7f ON public.workspace_member_invites USING btree (updated_by_id);


--
-- Name: workspace_member_invite_workspace_id_d935b364; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_invite_workspace_id_d935b364 ON public.workspace_member_invites USING btree (workspace_id);


--
-- Name: workspace_member_member_id_824f5497; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_member_id_824f5497 ON public.workspace_members USING btree (member_id);


--
-- Name: workspace_member_unique_workspace_member_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX workspace_member_unique_workspace_member_when_deleted_at_null ON public.workspace_members USING btree (workspace_id, member_id) WHERE (deleted_at IS NULL);


--
-- Name: workspace_member_updated_by_id_1cec0062; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_updated_by_id_1cec0062 ON public.workspace_members USING btree (updated_by_id);


--
-- Name: workspace_member_workspace_id_33f66d4b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_member_workspace_id_33f66d4b ON public.workspace_members USING btree (workspace_id);


--
-- Name: workspace_owner_id_60a8bafc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_owner_id_60a8bafc ON public.workspaces USING btree (owner_id);


--
-- Name: workspace_slug_4d89d459_like; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_slug_4d89d459_like ON public.workspaces USING btree (slug varchar_pattern_ops);


--
-- Name: workspace_theme_unique_workspace_name_when_deleted_at_null; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX workspace_theme_unique_workspace_name_when_deleted_at_null ON public.workspace_themes USING btree (workspace_id, name) WHERE (deleted_at IS NULL);


--
-- Name: workspace_themes_actor_id_0e94172e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_themes_actor_id_0e94172e ON public.workspace_themes USING btree (actor_id);


--
-- Name: workspace_themes_created_by_id_676e2655; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_themes_created_by_id_676e2655 ON public.workspace_themes USING btree (created_by_id);


--
-- Name: workspace_themes_updated_by_id_bba863fe; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_themes_updated_by_id_bba863fe ON public.workspace_themes USING btree (updated_by_id);


--
-- Name: workspace_themes_workspace_id_d1bffad8; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_themes_workspace_id_d1bffad8 ON public.workspace_themes USING btree (workspace_id);


--
-- Name: workspace_updated_by_id_09d249ed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_updated_by_id_09d249ed ON public.workspaces USING btree (updated_by_id);


--
-- Name: workspace_user_home_preferences_unique_workspace_user_key_when_; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX workspace_user_home_preferences_unique_workspace_user_key_when_ ON public.workspace_home_preferences USING btree (workspace_id, user_id, key) WHERE (deleted_at IS NULL);


--
-- Name: workspace_user_links_created_by_id_b9ce7a5d; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_links_created_by_id_b9ce7a5d ON public.workspace_user_links USING btree (created_by_id);


--
-- Name: workspace_user_links_owner_id_37d99444; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_links_owner_id_37d99444 ON public.workspace_user_links USING btree (owner_id);


--
-- Name: workspace_user_links_project_id_045e0d53; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_links_project_id_045e0d53 ON public.workspace_user_links USING btree (project_id);


--
-- Name: workspace_user_links_updated_by_id_bd0b017f; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_links_updated_by_id_bd0b017f ON public.workspace_user_links USING btree (updated_by_id);


--
-- Name: workspace_user_links_workspace_id_1b0a8e22; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_links_workspace_id_1b0a8e22 ON public.workspace_user_links USING btree (workspace_id);


--
-- Name: workspace_user_preferences_created_by_id_2d566570; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_preferences_created_by_id_2d566570 ON public.workspace_user_preferences USING btree (created_by_id);


--
-- Name: workspace_user_preferences_unique_workspace_user_key_when_delet; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX workspace_user_preferences_unique_workspace_user_key_when_delet ON public.workspace_user_preferences USING btree (workspace_id, user_id, key) WHERE (deleted_at IS NULL);


--
-- Name: workspace_user_preferences_updated_by_id_65fed266; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_preferences_updated_by_id_65fed266 ON public.workspace_user_preferences USING btree (updated_by_id);


--
-- Name: workspace_user_preferences_user_id_0ba5007a; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_preferences_user_id_0ba5007a ON public.workspace_user_preferences USING btree (user_id);


--
-- Name: workspace_user_preferences_workspace_id_a345adde; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_preferences_workspace_id_a345adde ON public.workspace_user_preferences USING btree (workspace_id);


--
-- Name: workspace_user_properties_created_by_id_6d8d1c4e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_properties_created_by_id_6d8d1c4e ON public.workspace_user_properties USING btree (created_by_id);


--
-- Name: workspace_user_properties_unique_workspace_user_when_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX workspace_user_properties_unique_workspace_user_when_deleted_at ON public.workspace_user_properties USING btree (workspace_id, user_id) WHERE (deleted_at IS NULL);


--
-- Name: workspace_user_properties_updated_by_id_910a2cc5; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_properties_updated_by_id_910a2cc5 ON public.workspace_user_properties USING btree (updated_by_id);


--
-- Name: workspace_user_properties_user_id_b1079e07; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_properties_user_id_b1079e07 ON public.workspace_user_properties USING btree (user_id);


--
-- Name: workspace_user_properties_workspace_id_1dc3e2a6; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspace_user_properties_workspace_id_1dc3e2a6 ON public.workspace_user_properties USING btree (workspace_id);


--
-- Name: workspaces_logo_asset_id_a784bb00; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX workspaces_logo_asset_id_a784bb00 ON public.workspaces USING btree (logo_asset_id);


--
-- Name: accounts accounts_user_id_7f1e1f1e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_user_id_7f1e1f1e_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: analytic_views analytic_views_created_by_id_1b3ca0a9_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytic_views
    ADD CONSTRAINT analytic_views_created_by_id_1b3ca0a9_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: analytic_views analytic_views_updated_by_id_b6d827e1_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytic_views
    ADD CONSTRAINT analytic_views_updated_by_id_b6d827e1_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: analytic_views analytic_views_workspace_id_ca6e5c0b_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.analytic_views
    ADD CONSTRAINT analytic_views_workspace_id_ca6e5c0b_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: api_activity_logs api_activity_logs_created_by_id_7f5c4ca8_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_activity_logs
    ADD CONSTRAINT api_activity_logs_created_by_id_7f5c4ca8_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: api_activity_logs api_activity_logs_updated_by_id_9ba0d417_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_activity_logs
    ADD CONSTRAINT api_activity_logs_updated_by_id_9ba0d417_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: api_tokens api_tokens_created_by_id_441e3d24_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_created_by_id_441e3d24_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: api_tokens api_tokens_updated_by_id_bcd544cf_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_updated_by_id_bcd544cf_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: api_tokens api_tokens_user_id_2db24e1c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_user_id_2db24e1c_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: api_tokens api_tokens_workspace_id_6791c7bd_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_workspace_id_6791c7bd_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: changelogs changelogs_created_by_id_16dd944a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelogs
    ADD CONSTRAINT changelogs_created_by_id_16dd944a_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: changelogs changelogs_updated_by_id_e0989861_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.changelogs
    ADD CONSTRAINT changelogs_updated_by_id_e0989861_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: comment_reactions comment_reactions_actor_id_21219e9c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_actor_id_21219e9c_fk_users_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: comment_reactions comment_reactions_comment_id_87c59446_fk_issue_comments_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_comment_id_87c59446_fk_issue_comments_id FOREIGN KEY (comment_id) REFERENCES public.issue_comments(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: comment_reactions comment_reactions_created_by_id_9aeb43c4_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_created_by_id_9aeb43c4_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: comment_reactions comment_reactions_project_id_ab9114b4_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_project_id_ab9114b4_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: comment_reactions comment_reactions_updated_by_id_c74c9bbd_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_updated_by_id_c74c9bbd_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: comment_reactions comment_reactions_workspace_id_b614ca4f_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reactions
    ADD CONSTRAINT comment_reactions_workspace_id_b614ca4f_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycles cycle_created_by_id_78e43b79_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycles
    ADD CONSTRAINT cycle_created_by_id_78e43b79_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_issues cycle_issue_created_by_id_30b27539_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issue_created_by_id_30b27539_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_issues cycle_issue_cycle_id_ec681215_fk_cycle_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issue_cycle_id_ec681215_fk_cycle_id FOREIGN KEY (cycle_id) REFERENCES public.cycles(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_issues cycle_issue_project_id_6ad3257a_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issue_project_id_6ad3257a_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_issues cycle_issue_updated_by_id_cb4516f2_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issue_updated_by_id_cb4516f2_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_issues cycle_issue_workspace_id_1d77330e_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issue_workspace_id_1d77330e_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_issues cycle_issues_issue_id_2d5ac97f_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_issues
    ADD CONSTRAINT cycle_issues_issue_id_2d5ac97f_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycles cycle_owned_by_id_5456a4d1_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycles
    ADD CONSTRAINT cycle_owned_by_id_5456a4d1_fk_user_id FOREIGN KEY (owned_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycles cycle_project_id_0b590349_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycles
    ADD CONSTRAINT cycle_project_id_0b590349_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycles cycle_updated_by_id_93baee43_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycles
    ADD CONSTRAINT cycle_updated_by_id_93baee43_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_user_properties cycle_user_properties_created_by_id_501f371c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_created_by_id_501f371c_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_user_properties cycle_user_properties_cycle_id_1f8bdf35_fk_cycles_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_cycle_id_1f8bdf35_fk_cycles_id FOREIGN KEY (cycle_id) REFERENCES public.cycles(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_user_properties cycle_user_properties_project_id_4efc0f07_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_project_id_4efc0f07_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_user_properties cycle_user_properties_updated_by_id_1b5ac27b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_updated_by_id_1b5ac27b_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_user_properties cycle_user_properties_user_id_9e9ef97d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_user_id_9e9ef97d_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycle_user_properties cycle_user_properties_workspace_id_62d65d71_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycle_user_properties
    ADD CONSTRAINT cycle_user_properties_workspace_id_62d65d71_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: cycles cycle_workspace_id_a199e8e1_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cycles
    ADD CONSTRAINT cycle_workspace_id_a199e8e1_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: deploy_boards deploy_boards_created_by_id_149dff93_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_created_by_id_149dff93_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: deploy_boards deploy_boards_intake_id_76a6470a_fk_intakes_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_intake_id_76a6470a_fk_intakes_id FOREIGN KEY (intake_id) REFERENCES public.intakes(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: deploy_boards deploy_boards_project_id_cfc792a1_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_project_id_cfc792a1_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: deploy_boards deploy_boards_updated_by_id_db7ae24f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_updated_by_id_db7ae24f_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: deploy_boards deploy_boards_workspace_id_fcf03158_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.deploy_boards
    ADD CONSTRAINT deploy_boards_workspace_id_fcf03158_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: description_versions description_versions_created_by_id_6633a3de_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.description_versions
    ADD CONSTRAINT description_versions_created_by_id_6633a3de_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: description_versions description_versions_description_id_dc7f19b6_fk_descriptions_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.description_versions
    ADD CONSTRAINT description_versions_description_id_dc7f19b6_fk_descriptions_id FOREIGN KEY (description_id) REFERENCES public.descriptions(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: description_versions description_versions_project_id_1a6c9aa9_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.description_versions
    ADD CONSTRAINT description_versions_project_id_1a6c9aa9_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: description_versions description_versions_updated_by_id_8b5179ae_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.description_versions
    ADD CONSTRAINT description_versions_updated_by_id_8b5179ae_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: description_versions description_versions_workspace_id_52857186_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.description_versions
    ADD CONSTRAINT description_versions_workspace_id_52857186_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: descriptions descriptions_created_by_id_b88ab399_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.descriptions
    ADD CONSTRAINT descriptions_created_by_id_b88ab399_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: descriptions descriptions_project_id_8f46180b_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.descriptions
    ADD CONSTRAINT descriptions_project_id_8f46180b_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: descriptions descriptions_updated_by_id_af519c4d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.descriptions
    ADD CONSTRAINT descriptions_updated_by_id_af519c4d_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: descriptions descriptions_workspace_id_767279bf_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.descriptions
    ADD CONSTRAINT descriptions_workspace_id_767279bf_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: device_sessions device_sessions_created_by_id_920a3bd5_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_created_by_id_920a3bd5_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: device_sessions device_sessions_device_id_a42b2ada_fk_devices_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_device_id_a42b2ada_fk_devices_id FOREIGN KEY (device_id) REFERENCES public.devices(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: device_sessions device_sessions_session_id_5382b02b_fk_sessions_session_key; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_session_id_5382b02b_fk_sessions_session_key FOREIGN KEY (session_id) REFERENCES public.sessions(session_key) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: device_sessions device_sessions_updated_by_id_d0bd0c76_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.device_sessions
    ADD CONSTRAINT device_sessions_updated_by_id_d0bd0c76_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: devices devices_created_by_id_410a755b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices
    ADD CONSTRAINT devices_created_by_id_410a755b_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: devices devices_updated_by_id_ee20dc3c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices
    ADD CONSTRAINT devices_updated_by_id_ee20dc3c_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: devices devices_user_id_9a5cca49_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.devices
    ADD CONSTRAINT devices_user_id_9a5cca49_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_assignees draft_issue_assignee_draft_issue_id_70827be2_fk_draft_iss; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignee_draft_issue_id_70827be2_fk_draft_iss FOREIGN KEY (draft_issue_id) REFERENCES public.draft_issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_assignees draft_issue_assignees_assignee_id_9cc52f9d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_assignee_id_9cc52f9d_fk_users_id FOREIGN KEY (assignee_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_assignees draft_issue_assignees_created_by_id_c25d4bde_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_created_by_id_c25d4bde_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_assignees draft_issue_assignees_project_id_c87dd571_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_project_id_c87dd571_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_assignees draft_issue_assignees_updated_by_id_16dbb5e0_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_updated_by_id_16dbb5e0_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_assignees draft_issue_assignees_workspace_id_e28a98e9_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_assignees
    ADD CONSTRAINT draft_issue_assignees_workspace_id_e28a98e9_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_cycles draft_issue_cycles_created_by_id_e56335c8_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_created_by_id_e56335c8_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_cycles draft_issue_cycles_cycle_id_b214e11f_fk_cycles_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_cycle_id_b214e11f_fk_cycles_id FOREIGN KEY (cycle_id) REFERENCES public.cycles(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_cycles draft_issue_cycles_draft_issue_id_ed45e8a2_fk_draft_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_draft_issue_id_ed45e8a2_fk_draft_issues_id FOREIGN KEY (draft_issue_id) REFERENCES public.draft_issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_cycles draft_issue_cycles_project_id_dc5d1ff6_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_project_id_dc5d1ff6_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_cycles draft_issue_cycles_updated_by_id_518a23ab_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_updated_by_id_518a23ab_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_cycles draft_issue_cycles_workspace_id_4fd0aa0c_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_cycles
    ADD CONSTRAINT draft_issue_cycles_workspace_id_4fd0aa0c_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_labels draft_issue_labels_created_by_id_88217eef_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_created_by_id_88217eef_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_labels draft_issue_labels_draft_issue_id_339d4c2b_fk_draft_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_draft_issue_id_339d4c2b_fk_draft_issues_id FOREIGN KEY (draft_issue_id) REFERENCES public.draft_issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_labels draft_issue_labels_label_id_b9b001a5_fk_labels_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_label_id_b9b001a5_fk_labels_id FOREIGN KEY (label_id) REFERENCES public.labels(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_labels draft_issue_labels_project_id_16f9ba0a_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_project_id_16f9ba0a_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_labels draft_issue_labels_updated_by_id_edac537c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_updated_by_id_edac537c_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_labels draft_issue_labels_workspace_id_489a9873_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_labels
    ADD CONSTRAINT draft_issue_labels_workspace_id_489a9873_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_modules draft_issue_modules_created_by_id_95ec4247_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_created_by_id_95ec4247_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_modules draft_issue_modules_draft_issue_id_eb470383_fk_draft_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_draft_issue_id_eb470383_fk_draft_issues_id FOREIGN KEY (draft_issue_id) REFERENCES public.draft_issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_modules draft_issue_modules_module_id_4d3f477a_fk_modules_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_module_id_4d3f477a_fk_modules_id FOREIGN KEY (module_id) REFERENCES public.modules(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_modules draft_issue_modules_project_id_c32eadab_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_project_id_c32eadab_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_modules draft_issue_modules_updated_by_id_18548965_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_updated_by_id_18548965_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issue_modules draft_issue_modules_workspace_id_536c335a_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issue_modules
    ADD CONSTRAINT draft_issue_modules_workspace_id_536c335a_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_created_by_id_aedba72a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_created_by_id_aedba72a_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_estimate_point_id_9e333189_fk_estimate_points_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_estimate_point_id_9e333189_fk_estimate_points_id FOREIGN KEY (estimate_point_id) REFERENCES public.estimate_points(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_parent_id_eee6ec32_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_parent_id_eee6ec32_fk_issues_id FOREIGN KEY (parent_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_project_id_784a560c_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_project_id_784a560c_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_state_id_94f28f5a_fk_states_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_state_id_94f28f5a_fk_states_id FOREIGN KEY (state_id) REFERENCES public.states(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_type_id_7a62fe34_fk_issue_types_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_type_id_7a62fe34_fk_issue_types_id FOREIGN KEY (type_id) REFERENCES public.issue_types(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_updated_by_id_1ca3cd4e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_updated_by_id_1ca3cd4e_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: draft_issues draft_issues_workspace_id_9d8512c8_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_issues
    ADD CONSTRAINT draft_issues_workspace_id_9d8512c8_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: email_notification_logs email_notification_logs_created_by_id_6faff587_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_notification_logs
    ADD CONSTRAINT email_notification_logs_created_by_id_6faff587_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: email_notification_logs email_notification_logs_receiver_id_7c7d2e13_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_notification_logs
    ADD CONSTRAINT email_notification_logs_receiver_id_7c7d2e13_fk_users_id FOREIGN KEY (receiver_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: email_notification_logs email_notification_logs_triggered_by_id_b551e727_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_notification_logs
    ADD CONSTRAINT email_notification_logs_triggered_by_id_b551e727_fk_users_id FOREIGN KEY (triggered_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: email_notification_logs email_notification_logs_updated_by_id_5d99c798_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_notification_logs
    ADD CONSTRAINT email_notification_logs_updated_by_id_5d99c798_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimate_points estimate_points_created_by_id_d1b04bd9_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimate_points
    ADD CONSTRAINT estimate_points_created_by_id_d1b04bd9_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimate_points estimate_points_estimate_id_4b4cb706_fk_estimates_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimate_points
    ADD CONSTRAINT estimate_points_estimate_id_4b4cb706_fk_estimates_id FOREIGN KEY (estimate_id) REFERENCES public.estimates(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimate_points estimate_points_project_id_ba9bcb2c_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimate_points
    ADD CONSTRAINT estimate_points_project_id_ba9bcb2c_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimate_points estimate_points_updated_by_id_a1da94e1_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimate_points
    ADD CONSTRAINT estimate_points_updated_by_id_a1da94e1_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimate_points estimate_points_workspace_id_96fc4f92_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimate_points
    ADD CONSTRAINT estimate_points_workspace_id_96fc4f92_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimates estimates_created_by_id_7e401493_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimates
    ADD CONSTRAINT estimates_created_by_id_7e401493_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimates estimates_project_id_7f195a41_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimates
    ADD CONSTRAINT estimates_project_id_7f195a41_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimates estimates_updated_by_id_b3fcfb1d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimates
    ADD CONSTRAINT estimates_updated_by_id_b3fcfb1d_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: estimates estimates_workspace_id_718811eb_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.estimates
    ADD CONSTRAINT estimates_workspace_id_718811eb_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: exporters exporters_created_by_id_44e1d9b3_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exporters
    ADD CONSTRAINT exporters_created_by_id_44e1d9b3_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: exporters exporters_initiated_by_id_d51f7552_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exporters
    ADD CONSTRAINT exporters_initiated_by_id_d51f7552_fk_users_id FOREIGN KEY (initiated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: exporters exporters_updated_by_id_d2572861_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exporters
    ADD CONSTRAINT exporters_updated_by_id_d2572861_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: exporters exporters_workspace_id_11a04317_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exporters
    ADD CONSTRAINT exporters_workspace_id_11a04317_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_asset_created_by_id_966942a0_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_asset_created_by_id_966942a0_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_asset_updated_by_id_d6aaf4f0_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_asset_updated_by_id_d6aaf4f0_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_comment_id_35d4ecaf_fk_issue_comments_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_comment_id_35d4ecaf_fk_issue_comments_id FOREIGN KEY (comment_id) REFERENCES public.issue_comments(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_draft_issue_id_52633145_fk_draft_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_draft_issue_id_52633145_fk_draft_issues_id FOREIGN KEY (draft_issue_id) REFERENCES public.draft_issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_issue_id_cfe87d6c_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_issue_id_cfe87d6c_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_page_id_64c753d1_fk_pages_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_page_id_64c753d1_fk_pages_id FOREIGN KEY (page_id) REFERENCES public.pages(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_project_id_ebd5c0d8_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_project_id_ebd5c0d8_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_user_id_ce1818dc_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_user_id_ce1818dc_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: file_assets file_assets_workspace_id_fa50b9c5_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_assets
    ADD CONSTRAINT file_assets_workspace_id_fa50b9c5_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_comment_syncs github_comment_syncs_comment_id_6feec6d1_fk_issue_comments_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_comment_id_6feec6d1_fk_issue_comments_id FOREIGN KEY (comment_id) REFERENCES public.issue_comments(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_comment_syncs github_comment_syncs_created_by_id_b1ef2517_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_created_by_id_b1ef2517_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_comment_syncs github_comment_syncs_issue_sync_id_5e738eb5_fk_github_is; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_issue_sync_id_5e738eb5_fk_github_is FOREIGN KEY (issue_sync_id) REFERENCES public.github_issue_syncs(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_comment_syncs github_comment_syncs_project_id_6d199ace_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_project_id_6d199ace_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_comment_syncs github_comment_syncs_updated_by_id_bb05c066_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_updated_by_id_bb05c066_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_comment_syncs github_comment_syncs_workspace_id_b54528c8_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_comment_syncs
    ADD CONSTRAINT github_comment_syncs_workspace_id_b54528c8_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_issue_syncs github_issue_syncs_created_by_id_d02b7c56_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_created_by_id_d02b7c56_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_issue_syncs github_issue_syncs_issue_id_450cb083_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_issue_id_450cb083_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_issue_syncs github_issue_syncs_project_id_4609ad0c_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_project_id_4609ad0c_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_issue_syncs github_issue_syncs_repository_sync_id_ba0d4de4_fk_github_re; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_repository_sync_id_ba0d4de4_fk_github_re FOREIGN KEY (repository_sync_id) REFERENCES public.github_repository_syncs(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_issue_syncs github_issue_syncs_updated_by_id_e9cd6f86_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_updated_by_id_e9cd6f86_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_issue_syncs github_issue_syncs_workspace_id_eae020ad_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_issue_syncs
    ADD CONSTRAINT github_issue_syncs_workspace_id_eae020ad_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repositories github_repositories_created_by_id_104fa685_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repositories
    ADD CONSTRAINT github_repositories_created_by_id_104fa685_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repositories github_repositories_project_id_65c546bb_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repositories
    ADD CONSTRAINT github_repositories_project_id_65c546bb_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repositories github_repositories_updated_by_id_8aa4d772_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repositories
    ADD CONSTRAINT github_repositories_updated_by_id_8aa4d772_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repositories github_repositories_workspace_id_c4de7326_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repositories
    ADD CONSTRAINT github_repositories_workspace_id_c4de7326_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_sy_repository_id_ead52404_fk_github_re; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_sy_repository_id_ead52404_fk_github_re FOREIGN KEY (repository_id) REFERENCES public.github_repositories(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_sy_workspace_integratio_62858398_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_sy_workspace_integratio_62858398_fk_workspace FOREIGN KEY (workspace_integration_id) REFERENCES public.workspace_integrations(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_syncs_actor_id_1fa689fe_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_actor_id_1fa689fe_fk_users_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_syncs_created_by_id_0df94495_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_created_by_id_0df94495_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_syncs_label_id_eb1e9bd7_fk_labels_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_label_id_eb1e9bd7_fk_labels_id FOREIGN KEY (label_id) REFERENCES public.labels(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_syncs_project_id_e7e8291e_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_project_id_e7e8291e_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_syncs_updated_by_id_07e9d065_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_updated_by_id_07e9d065_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: github_repository_syncs github_repository_syncs_workspace_id_4a22a8b8_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repository_syncs
    ADD CONSTRAINT github_repository_syncs_workspace_id_4a22a8b8_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: importers importers_created_by_id_7dd06433_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_created_by_id_7dd06433_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: importers importers_initiated_by_id_3cddbd23_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_initiated_by_id_3cddbd23_fk_users_id FOREIGN KEY (initiated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: importers importers_project_id_1f8b43ef_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_project_id_1f8b43ef_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: importers importers_token_id_c951e89f_fk_api_tokens_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_token_id_c951e89f_fk_api_tokens_id FOREIGN KEY (token_id) REFERENCES public.api_tokens(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: importers importers_updated_by_id_3915139e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_updated_by_id_3915139e_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: importers importers_workspace_id_795b8985_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.importers
    ADD CONSTRAINT importers_workspace_id_795b8985_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_created_by_id_483bce13_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_created_by_id_483bce13_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_duplicate_to_id_6cb8d961_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_duplicate_to_id_6cb8d961_fk_issues_id FOREIGN KEY (duplicate_to_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_intake_id_a04a7455_fk_intakes_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_intake_id_a04a7455_fk_intakes_id FOREIGN KEY (intake_id) REFERENCES public.intakes(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_issue_id_7d74b224_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_issue_id_7d74b224_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_project_id_5117a70b_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_project_id_5117a70b_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_updated_by_id_d1b2b70f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_updated_by_id_d1b2b70f_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intake_issues inbox_issues_workspace_id_4a61a7bd_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intake_issues
    ADD CONSTRAINT inbox_issues_workspace_id_4a61a7bd_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intakes inboxes_created_by_id_9f1cf5ec_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intakes
    ADD CONSTRAINT inboxes_created_by_id_9f1cf5ec_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intakes inboxes_project_id_a0135c66_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intakes
    ADD CONSTRAINT inboxes_project_id_a0135c66_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intakes inboxes_updated_by_id_69b7b3ae_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intakes
    ADD CONSTRAINT inboxes_updated_by_id_69b7b3ae_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: intakes inboxes_workspace_id_d6178865_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.intakes
    ADD CONSTRAINT inboxes_workspace_id_d6178865_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instance_admins instance_admins_created_by_id_7f4e03b4_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_admins
    ADD CONSTRAINT instance_admins_created_by_id_7f4e03b4_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instance_admins instance_admins_instance_id_66d1ba73_fk_instances_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_admins
    ADD CONSTRAINT instance_admins_instance_id_66d1ba73_fk_instances_id FOREIGN KEY (instance_id) REFERENCES public.instances(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instance_admins instance_admins_updated_by_id_b7800403_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_admins
    ADD CONSTRAINT instance_admins_updated_by_id_b7800403_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instance_admins instance_admins_user_id_cc6e9b62_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_admins
    ADD CONSTRAINT instance_admins_user_id_cc6e9b62_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instance_configurations instance_configurations_created_by_id_e683f3e5_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_configurations
    ADD CONSTRAINT instance_configurations_created_by_id_e683f3e5_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instance_configurations instance_configurations_updated_by_id_f0d7542e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instance_configurations
    ADD CONSTRAINT instance_configurations_updated_by_id_f0d7542e_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instances instances_created_by_id_c76e92ef_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instances
    ADD CONSTRAINT instances_created_by_id_c76e92ef_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: instances instances_updated_by_id_cce8fcdf_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.instances
    ADD CONSTRAINT instances_updated_by_id_cce8fcdf_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: integrations integrations_created_by_id_0b6edd52_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_created_by_id_0b6edd52_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: integrations integrations_updated_by_id_d6d00d15_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.integrations
    ADD CONSTRAINT integrations_updated_by_id_d6d00d15_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activities_issue_id_180e5662_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activities_issue_id_180e5662_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activity_actor_id_52fdd42d_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_actor_id_52fdd42d_fk_user_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activity_created_by_id_49516e3d_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_created_by_id_49516e3d_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activity_issue_comment_id_701f3c3c_fk_issue_comment_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_issue_comment_id_701f3c3c_fk_issue_comment_id FOREIGN KEY (issue_comment_id) REFERENCES public.issue_comments(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activity_project_id_d0ac2ccf_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_project_id_d0ac2ccf_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activity_updated_by_id_0075f9bd_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_updated_by_id_0075f9bd_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_activities issue_activity_workspace_id_65acaf73_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_activities
    ADD CONSTRAINT issue_activity_workspace_id_65acaf73_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_assignees issue_assignee_assignee_id_50f5c04e_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_assignee_id_50f5c04e_fk_user_id FOREIGN KEY (assignee_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_assignees issue_assignee_created_by_id_f693d43b_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_created_by_id_f693d43b_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_assignees issue_assignee_issue_id_72da08db_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_issue_id_72da08db_fk_issue_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_assignees issue_assignee_project_id_61c18bf2_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_project_id_61c18bf2_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_assignees issue_assignee_updated_by_id_c54088aa_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_updated_by_id_c54088aa_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_assignees issue_assignee_workspace_id_9aad55b7_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_assignees
    ADD CONSTRAINT issue_assignee_workspace_id_9aad55b7_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_attachments issue_attachments_created_by_id_87be05bb_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_attachments
    ADD CONSTRAINT issue_attachments_created_by_id_87be05bb_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_attachments issue_attachments_issue_id_0faf88bf_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_attachments
    ADD CONSTRAINT issue_attachments_issue_id_0faf88bf_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_attachments issue_attachments_project_id_a95fe706_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_attachments
    ADD CONSTRAINT issue_attachments_project_id_a95fe706_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_attachments issue_attachments_updated_by_id_47dceec1_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_attachments
    ADD CONSTRAINT issue_attachments_updated_by_id_47dceec1_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_attachments issue_attachments_workspace_id_c456a532_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_attachments
    ADD CONSTRAINT issue_attachments_workspace_id_c456a532_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_blockers issue_blocker_block_id_5d15a701_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_block_id_5d15a701_fk_issue_id FOREIGN KEY (block_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_blockers issue_blocker_blocked_by_id_a138af71_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_blocked_by_id_a138af71_fk_issue_id FOREIGN KEY (blocked_by_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_blockers issue_blocker_created_by_id_0d19f6ea_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_created_by_id_0d19f6ea_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_blockers issue_blocker_project_id_380bd100_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_project_id_380bd100_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_blockers issue_blocker_updated_by_id_4af87d63_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_updated_by_id_4af87d63_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_blockers issue_blocker_workspace_id_419a1c71_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_blockers
    ADD CONSTRAINT issue_blocker_workspace_id_419a1c71_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comment_actor_id_d312315b_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_actor_id_d312315b_fk_user_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comment_created_by_id_0765f239_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_created_by_id_0765f239_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comment_issue_id_d0195e35_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_issue_id_d0195e35_fk_issue_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comment_project_id_db37c105_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_project_id_db37c105_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comment_updated_by_id_96cfb86e_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_updated_by_id_96cfb86e_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comment_workspace_id_3f7969ec_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comment_workspace_id_3f7969ec_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comments_description_id_0cb72512_fk_descriptions_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comments_description_id_0cb72512_fk_descriptions_id FOREIGN KEY (description_id) REFERENCES public.descriptions(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_comments issue_comments_parent_id_d8db10b1_fk_issue_comments_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_comments
    ADD CONSTRAINT issue_comments_parent_id_d8db10b1_fk_issue_comments_id FOREIGN KEY (parent_id) REFERENCES public.issue_comments(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issue_created_by_id_8f0ae62b_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_created_by_id_8f0ae62b_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_description_versions issue_description_ve_workspace_id_88e930f9_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_ve_workspace_id_88e930f9_fk_workspace FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_description_versions issue_description_versions_created_by_id_3f7e62a1_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_versions_created_by_id_3f7e62a1_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_description_versions issue_description_versions_issue_id_c8baa13e_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_versions_issue_id_c8baa13e_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_description_versions issue_description_versions_owned_by_id_0effe4d0_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_versions_owned_by_id_0effe4d0_fk_users_id FOREIGN KEY (owned_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_description_versions issue_description_versions_project_id_536b23ef_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_versions_project_id_536b23ef_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_description_versions issue_description_versions_updated_by_id_6530365d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_description_versions
    ADD CONSTRAINT issue_description_versions_updated_by_id_6530365d_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_labels issue_label_created_by_id_94075315_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_created_by_id_94075315_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_labels issue_label_issue_id_0f252e52_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_issue_id_0f252e52_fk_issue_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_labels issue_label_label_id_5f22777f_fk_label_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_label_id_5f22777f_fk_label_id FOREIGN KEY (label_id) REFERENCES public.labels(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_labels issue_label_project_id_eaa2ba39_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_project_id_eaa2ba39_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_labels issue_label_updated_by_id_a97a6733_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_updated_by_id_a97a6733_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_labels issue_label_workspace_id_b5b1faac_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_labels
    ADD CONSTRAINT issue_label_workspace_id_b5b1faac_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_links issue_links_created_by_id_5e4aa092_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_links
    ADD CONSTRAINT issue_links_created_by_id_5e4aa092_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_links issue_links_issue_id_7032881f_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_links
    ADD CONSTRAINT issue_links_issue_id_7032881f_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_links issue_links_project_id_63d6e9ce_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_links
    ADD CONSTRAINT issue_links_project_id_63d6e9ce_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_links issue_links_updated_by_id_a771cce4_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_links
    ADD CONSTRAINT issue_links_updated_by_id_a771cce4_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_links issue_links_workspace_id_ff9038e7_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_links
    ADD CONSTRAINT issue_links_workspace_id_ff9038e7_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_mentions issue_mentions_created_by_id_eb44759e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_created_by_id_eb44759e_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_mentions issue_mentions_issue_id_d8821107_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_issue_id_d8821107_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_mentions issue_mentions_mention_id_cf1b9346_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_mention_id_cf1b9346_fk_users_id FOREIGN KEY (mention_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_mentions issue_mentions_project_id_d0cccdf5_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_project_id_d0cccdf5_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_mentions issue_mentions_updated_by_id_c62106d3_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_updated_by_id_c62106d3_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_mentions issue_mentions_workspace_id_4ca59d05_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_mentions
    ADD CONSTRAINT issue_mentions_workspace_id_4ca59d05_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issue_parent_id_ce8d76ba_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_parent_id_ce8d76ba_fk_issue_id FOREIGN KEY (parent_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issue_project_id_fea0fc80_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_project_id_fea0fc80_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_user_properties issue_property_created_by_id_8e92131c_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_property_created_by_id_8e92131c_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_user_properties issue_property_project_id_30e7de7b_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_property_project_id_30e7de7b_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_user_properties issue_property_updated_by_id_ff158d4d_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_property_updated_by_id_ff158d4d_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_user_properties issue_property_user_id_0b1d1c8f_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_property_user_id_0b1d1c8f_fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_user_properties issue_property_workspace_id_17860d65_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_user_properties
    ADD CONSTRAINT issue_property_workspace_id_17860d65_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_reactions issue_reactions_actor_id_5f5b8303_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_actor_id_5f5b8303_fk_users_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_reactions issue_reactions_created_by_id_3953b7de_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_created_by_id_3953b7de_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_reactions issue_reactions_issue_id_2c324bae_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_issue_id_2c324bae_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_reactions issue_reactions_project_id_8708ecaf_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_project_id_8708ecaf_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_reactions issue_reactions_updated_by_id_4069af90_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_updated_by_id_4069af90_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_reactions issue_reactions_workspace_id_bd8d7550_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reactions
    ADD CONSTRAINT issue_reactions_workspace_id_bd8d7550_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_relations issue_relations_created_by_id_854d07e7_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_created_by_id_854d07e7_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_relations issue_relations_issue_id_e1db6f72_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_issue_id_e1db6f72_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_relations issue_relations_project_id_15350161_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_project_id_15350161_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_relations issue_relations_related_issue_id_e1ea44a7_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_related_issue_id_e1ea44a7_fk_issues_id FOREIGN KEY (related_issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_relations issue_relations_updated_by_id_3dfa850f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_updated_by_id_3dfa850f_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_relations issue_relations_workspace_id_00b50e90_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_relations
    ADD CONSTRAINT issue_relations_workspace_id_00b50e90_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_sequences issue_sequence_created_by_id_59270506_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_sequences
    ADD CONSTRAINT issue_sequence_created_by_id_59270506_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_sequences issue_sequence_issue_id_16e9f00f_fk_issue_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_sequences
    ADD CONSTRAINT issue_sequence_issue_id_16e9f00f_fk_issue_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_sequences issue_sequence_project_id_ce882e85_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_sequences
    ADD CONSTRAINT issue_sequence_project_id_ce882e85_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_sequences issue_sequence_updated_by_id_310c8dd3_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_sequences
    ADD CONSTRAINT issue_sequence_updated_by_id_310c8dd3_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_sequences issue_sequence_workspace_id_0d3f0fd4_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_sequences
    ADD CONSTRAINT issue_sequence_workspace_id_0d3f0fd4_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issue_state_id_1a65560d_fk_state_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_state_id_1a65560d_fk_state_id FOREIGN KEY (state_id) REFERENCES public.states(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_subscribers issue_subscribers_created_by_id_b6ea0157_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_created_by_id_b6ea0157_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_subscribers issue_subscribers_issue_id_85cf2093_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_issue_id_85cf2093_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_subscribers issue_subscribers_project_id_cf48d75f_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_project_id_cf48d75f_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_subscribers issue_subscribers_subscriber_id_2d89c988_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_subscriber_id_2d89c988_fk_users_id FOREIGN KEY (subscriber_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_subscribers issue_subscribers_updated_by_id_1bfc2f55_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_updated_by_id_1bfc2f55_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_subscribers issue_subscribers_workspace_id_96afa91f_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscribers
    ADD CONSTRAINT issue_subscribers_workspace_id_96afa91f_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_types issue_types_created_by_id_48764f53_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_types
    ADD CONSTRAINT issue_types_created_by_id_48764f53_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_types issue_types_updated_by_id_4919203b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_types
    ADD CONSTRAINT issue_types_updated_by_id_4919203b_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_types issue_types_workspace_id_591c6f3b_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_types
    ADD CONSTRAINT issue_types_workspace_id_591c6f3b_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issue_updated_by_id_f1261863_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_updated_by_id_f1261863_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_activity_id_b1872ffc_fk_issue_activities_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_activity_id_b1872ffc_fk_issue_activities_id FOREIGN KEY (activity_id) REFERENCES public.issue_activities(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_created_by_id_a782830a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_created_by_id_a782830a_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_issue_id_25cf001c_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_issue_id_25cf001c_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_owned_by_id_7586378d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_owned_by_id_7586378d_fk_users_id FOREIGN KEY (owned_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_project_id_a069ad03_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_project_id_a069ad03_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_updated_by_id_dcae6dd2_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_updated_by_id_dcae6dd2_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_versions issue_versions_workspace_id_b8c48b7c_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_versions
    ADD CONSTRAINT issue_versions_workspace_id_b8c48b7c_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_views issue_views_created_by_id_0d2e456b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_views
    ADD CONSTRAINT issue_views_created_by_id_0d2e456b_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_views issue_views_owned_by_id_5e261e5d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_views
    ADD CONSTRAINT issue_views_owned_by_id_5e261e5d_fk_users_id FOREIGN KEY (owned_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_views issue_views_project_id_55ee009f_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_views
    ADD CONSTRAINT issue_views_project_id_55ee009f_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_views issue_views_updated_by_id_28cd9870_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_views
    ADD CONSTRAINT issue_views_updated_by_id_28cd9870_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_views issue_views_workspace_id_8785e03d_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_views
    ADD CONSTRAINT issue_views_workspace_id_8785e03d_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_votes issue_votes_actor_id_525cab61_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_actor_id_525cab61_fk_users_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_votes issue_votes_created_by_id_86adcf5c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_created_by_id_86adcf5c_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_votes issue_votes_issue_id_07a61ecb_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_issue_id_07a61ecb_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_votes issue_votes_project_id_b649f55b_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_project_id_b649f55b_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_votes issue_votes_updated_by_id_9e2a6cdc_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_updated_by_id_9e2a6cdc_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issue_votes issue_votes_workspace_id_a3e91a6b_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_votes
    ADD CONSTRAINT issue_votes_workspace_id_a3e91a6b_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issue_workspace_id_c84878c1_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issue_workspace_id_c84878c1_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issues_estimate_point_id_a6822abe_fk_estimate_points_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issues_estimate_point_id_a6822abe_fk_estimate_points_id FOREIGN KEY (estimate_point_id) REFERENCES public.estimate_points(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: issues issues_type_id_a4710b19_fk_issue_types_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issues
    ADD CONSTRAINT issues_type_id_a4710b19_fk_issue_types_id FOREIGN KEY (type_id) REFERENCES public.issue_types(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: labels label_created_by_id_aa6ffcfa_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.labels
    ADD CONSTRAINT label_created_by_id_aa6ffcfa_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: labels label_parent_id_7a853296_fk_label_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.labels
    ADD CONSTRAINT label_parent_id_7a853296_fk_label_id FOREIGN KEY (parent_id) REFERENCES public.labels(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: labels label_updated_by_id_894a5464_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.labels
    ADD CONSTRAINT label_updated_by_id_894a5464_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: labels label_workspace_id_c4c9ae5a_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.labels
    ADD CONSTRAINT label_workspace_id_c4c9ae5a_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: labels labels_project_id_cf57a802_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.labels
    ADD CONSTRAINT labels_project_id_cf57a802_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: modules module_created_by_id_ff7a5866_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT module_created_by_id_ff7a5866_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_issues module_issues_created_by_id_de0b995a_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_created_by_id_de0b995a_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_issues module_issues_issue_id_7caa908b_fk_issues_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_issue_id_7caa908b_fk_issues_id FOREIGN KEY (issue_id) REFERENCES public.issues(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_issues module_issues_module_id_74e0ed5a_fk_module_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_module_id_74e0ed5a_fk_module_id FOREIGN KEY (module_id) REFERENCES public.modules(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_issues module_issues_project_id_59836d1e_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_project_id_59836d1e_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_issues module_issues_updated_by_id_46dbf724_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_updated_by_id_46dbf724_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_issues module_issues_workspace_id_6bf85201_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_issues
    ADD CONSTRAINT module_issues_workspace_id_6bf85201_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: modules module_lead_id_04966630_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT module_lead_id_04966630_fk_user_id FOREIGN KEY (lead_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_links module_links_created_by_id_eaf6492f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_links
    ADD CONSTRAINT module_links_created_by_id_eaf6492f_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_links module_links_module_id_0fda3f8a_fk_modules_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_links
    ADD CONSTRAINT module_links_module_id_0fda3f8a_fk_modules_id FOREIGN KEY (module_id) REFERENCES public.modules(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_links module_links_project_id_f720bb79_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_links
    ADD CONSTRAINT module_links_project_id_f720bb79_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_links module_links_updated_by_id_4da419e7_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_links
    ADD CONSTRAINT module_links_updated_by_id_4da419e7_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_links module_links_workspace_id_0521c11c_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_links
    ADD CONSTRAINT module_links_workspace_id_0521c11c_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_members module_member_created_by_id_2ed84a65_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_created_by_id_2ed84a65_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_members module_member_member_id_928f473e_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_member_id_928f473e_fk_user_id FOREIGN KEY (member_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_members module_member_module_id_f00be7ef_fk_module_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_module_id_f00be7ef_fk_module_id FOREIGN KEY (module_id) REFERENCES public.modules(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_members module_member_project_id_ec8d2376_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_project_id_ec8d2376_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_members module_member_updated_by_id_a9046438_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_updated_by_id_a9046438_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_members module_member_workspace_id_f2f23c73_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_members
    ADD CONSTRAINT module_member_workspace_id_f2f23c73_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: modules module_project_id_da84b04f_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT module_project_id_da84b04f_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: modules module_updated_by_id_72ab6d5c_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT module_updated_by_id_72ab6d5c_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_user_properties module_user_properties_created_by_id_bdd98440_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_created_by_id_bdd98440_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_user_properties module_user_properties_module_id_e95b158a_fk_modules_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_module_id_e95b158a_fk_modules_id FOREIGN KEY (module_id) REFERENCES public.modules(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_user_properties module_user_properties_project_id_3c5a4972_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_project_id_3c5a4972_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_user_properties module_user_properties_updated_by_id_b7dafc77_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_updated_by_id_b7dafc77_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_user_properties module_user_properties_user_id_e83a1c2c_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_user_id_e83a1c2c_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: module_user_properties module_user_properties_workspace_id_ddaf807c_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.module_user_properties
    ADD CONSTRAINT module_user_properties_workspace_id_ddaf807c_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: modules module_workspace_id_0a826fef_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.modules
    ADD CONSTRAINT module_workspace_id_0a826fef_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: notifications notifications_created_by_id_b9c3f81b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_created_by_id_b9c3f81b_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: notifications notifications_project_id_e4d4f192_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_project_id_e4d4f192_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: notifications notifications_receiver_id_b708b2b0_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_receiver_id_b708b2b0_fk_users_id FOREIGN KEY (receiver_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: notifications notifications_triggered_by_id_31cdec21_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_triggered_by_id_31cdec21_fk_users_id FOREIGN KEY (triggered_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: notifications notifications_updated_by_id_8a651e96_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_updated_by_id_8a651e96_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: notifications notifications_workspace_id_b2f09ef7_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_workspace_id_b2f09ef7_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_labels page_labels_created_by_id_fbd942c0_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_labels
    ADD CONSTRAINT page_labels_created_by_id_fbd942c0_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_labels page_labels_label_id_05958e53_fk_labels_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_labels
    ADD CONSTRAINT page_labels_label_id_05958e53_fk_labels_id FOREIGN KEY (label_id) REFERENCES public.labels(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_labels page_labels_page_id_0e6cdb3d_fk_pages_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_labels
    ADD CONSTRAINT page_labels_page_id_0e6cdb3d_fk_pages_id FOREIGN KEY (page_id) REFERENCES public.pages(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_labels page_labels_updated_by_id_d9fddbff_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_labels
    ADD CONSTRAINT page_labels_updated_by_id_d9fddbff_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_labels page_labels_workspace_id_078bb01c_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_labels
    ADD CONSTRAINT page_labels_workspace_id_078bb01c_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_logs page_logs_created_by_id_4a295aec_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_logs
    ADD CONSTRAINT page_logs_created_by_id_4a295aec_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_logs page_logs_page_id_0e0d747d_fk_pages_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_logs
    ADD CONSTRAINT page_logs_page_id_0e0d747d_fk_pages_id FOREIGN KEY (page_id) REFERENCES public.pages(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_logs page_logs_updated_by_id_1995190b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_logs
    ADD CONSTRAINT page_logs_updated_by_id_1995190b_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_logs page_logs_workspace_id_be7bde64_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_logs
    ADD CONSTRAINT page_logs_workspace_id_be7bde64_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_versions page_versions_created_by_id_d660b13b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_versions
    ADD CONSTRAINT page_versions_created_by_id_d660b13b_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_versions page_versions_owned_by_id_6d9143db_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_versions
    ADD CONSTRAINT page_versions_owned_by_id_6d9143db_fk_users_id FOREIGN KEY (owned_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_versions page_versions_page_id_c46471da_fk_pages_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_versions
    ADD CONSTRAINT page_versions_page_id_c46471da_fk_pages_id FOREIGN KEY (page_id) REFERENCES public.pages(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_versions page_versions_updated_by_id_72d5e579_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_versions
    ADD CONSTRAINT page_versions_updated_by_id_72d5e579_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: page_versions page_versions_workspace_id_8330a200_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_versions
    ADD CONSTRAINT page_versions_workspace_id_8330a200_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: pages pages_created_by_id_d109a675_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_created_by_id_d109a675_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: pages pages_owned_by_id_bf50485f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_owned_by_id_bf50485f_fk_users_id FOREIGN KEY (owned_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: pages pages_parent_id_8b823409_fk_pages_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_parent_id_8b823409_fk_pages_id FOREIGN KEY (parent_id) REFERENCES public.pages(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: pages pages_updated_by_id_6c42de3e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_updated_by_id_6c42de3e_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: pages pages_workspace_id_c6c51010_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_workspace_id_c6c51010_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: profiles profiles_user_id_36580373_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_user_id_36580373_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects project_created_by_id_6cc13408_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT project_created_by_id_6cc13408_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects project_default_assignee_id_6ba45f90_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT project_default_assignee_id_6ba45f90_fk_user_id FOREIGN KEY (default_assignee_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_deploy_boards project_deploy_boards_created_by_id_2ea72f98_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_created_by_id_2ea72f98_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_deploy_boards project_deploy_boards_intake_id_36aa612d_fk_intakes_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_intake_id_36aa612d_fk_intakes_id FOREIGN KEY (intake_id) REFERENCES public.intakes(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_deploy_boards project_deploy_boards_project_id_49d887b2_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_project_id_49d887b2_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_deploy_boards project_deploy_boards_updated_by_id_290eb99e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_updated_by_id_290eb99e_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_deploy_boards project_deploy_boards_workspace_id_cd92f164_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_deploy_boards
    ADD CONSTRAINT project_deploy_boards_workspace_id_cd92f164_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_identifiers project_identifier_created_by_id_2b6f273a_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifier_created_by_id_2b6f273a_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_identifiers project_identifier_project_id_13de58a9_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifier_project_id_13de58a9_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_identifiers project_identifier_updated_by_id_1a00e2a0_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifier_updated_by_id_1a00e2a0_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_identifiers project_identifier_workspace_id_6024b517_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_identifiers
    ADD CONSTRAINT project_identifier_workspace_id_6024b517_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_issue_types project_issue_types_created_by_id_049cecfd_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_created_by_id_049cecfd_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_issue_types project_issue_types_issue_type_id_9494de9f_fk_issue_types_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_issue_type_id_9494de9f_fk_issue_types_id FOREIGN KEY (issue_type_id) REFERENCES public.issue_types(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_issue_types project_issue_types_project_id_ef6e52e4_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_project_id_ef6e52e4_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_issue_types project_issue_types_updated_by_id_b5998397_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_updated_by_id_b5998397_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_issue_types project_issue_types_workspace_id_ace3c5b5_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_issue_types
    ADD CONSTRAINT project_issue_types_workspace_id_ace3c5b5_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_members project_member_created_by_id_8b363306_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_member_created_by_id_8b363306_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_member_invites project_member_invite_created_by_id_a87df45c_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_member_invites
    ADD CONSTRAINT project_member_invite_created_by_id_a87df45c_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_member_invites project_member_invite_project_id_8fb7750e_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_member_invites
    ADD CONSTRAINT project_member_invite_project_id_8fb7750e_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_member_invites project_member_invite_updated_by_id_5aa55c96_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_member_invites
    ADD CONSTRAINT project_member_invite_updated_by_id_5aa55c96_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_member_invites project_member_invite_workspace_id_64e2dc4c_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_member_invites
    ADD CONSTRAINT project_member_invite_workspace_id_64e2dc4c_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_members project_member_member_id_9d6b126b_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_member_member_id_9d6b126b_fk_user_id FOREIGN KEY (member_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_members project_member_project_id_11ea1a9e_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_member_project_id_11ea1a9e_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_members project_member_updated_by_id_cf6aaac4_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_member_updated_by_id_cf6aaac4_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_members project_member_workspace_id_88bb9a97_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_member_workspace_id_88bb9a97_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_pages project_pages_created_by_id_b9d02062_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_created_by_id_b9d02062_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_pages project_pages_page_id_a0f54439_fk_pages_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_page_id_a0f54439_fk_pages_id FOREIGN KEY (page_id) REFERENCES public.pages(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_pages project_pages_project_id_376ba35a_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_project_id_376ba35a_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_pages project_pages_updated_by_id_b80bf0f4_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_updated_by_id_b80bf0f4_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_pages project_pages_workspace_id_13ed9e73_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_pages
    ADD CONSTRAINT project_pages_workspace_id_13ed9e73_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects project_project_lead_id_caf8e353_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT project_project_lead_id_caf8e353_fk_user_id FOREIGN KEY (project_lead_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_public_members project_public_members_created_by_id_c4c7c776_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_created_by_id_c4c7c776_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_public_members project_public_members_member_id_52f257f9_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_member_id_52f257f9_fk_users_id FOREIGN KEY (member_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_public_members project_public_members_project_id_2dfd893d_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_project_id_2dfd893d_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_public_members project_public_members_updated_by_id_c3e4d675_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_updated_by_id_c3e4d675_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_public_members project_public_members_workspace_id_ebfce110_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_public_members
    ADD CONSTRAINT project_public_members_workspace_id_ebfce110_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects project_updated_by_id_fe290525_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT project_updated_by_id_fe290525_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_webhooks project_webhooks_created_by_id_c3e4bfa3_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_created_by_id_c3e4bfa3_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_webhooks project_webhooks_project_id_bec3cf8c_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_project_id_bec3cf8c_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_webhooks project_webhooks_updated_by_id_a0183aeb_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_updated_by_id_a0183aeb_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_webhooks project_webhooks_webhook_id_da27c6a7_fk_webhooks_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_webhook_id_da27c6a7_fk_webhooks_id FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: project_webhooks project_webhooks_workspace_id_429ebf05_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_webhooks
    ADD CONSTRAINT project_webhooks_workspace_id_429ebf05_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects project_workspace_id_01764ff9_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT project_workspace_id_01764ff9_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects projects_cover_image_asset_id_e6636b92_fk_file_assets_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_cover_image_asset_id_e6636b92_fk_file_assets_id FOREIGN KEY (cover_image_asset_id) REFERENCES public.file_assets(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects projects_default_state_id_f13e8b95_fk_states_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_default_state_id_f13e8b95_fk_states_id FOREIGN KEY (default_state_id) REFERENCES public.states(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: projects projects_estimate_id_85c7b2ac_fk_estimates_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_estimate_id_85c7b2ac_fk_estimates_id FOREIGN KEY (estimate_id) REFERENCES public.estimates(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: slack_project_syncs slack_project_syncs_created_by_id_ec405a17_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_created_by_id_ec405a17_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: slack_project_syncs slack_project_syncs_project_id_016dc792_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_project_id_016dc792_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: slack_project_syncs slack_project_syncs_updated_by_id_152eb3b5_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_updated_by_id_152eb3b5_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: slack_project_syncs slack_project_syncs_workspace_id_d1822b06_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_workspace_id_d1822b06_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: slack_project_syncs slack_project_syncs_workspace_integratio_d89c9b40_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.slack_project_syncs
    ADD CONSTRAINT slack_project_syncs_workspace_integratio_d89c9b40_fk_workspace FOREIGN KEY (workspace_integration_id) REFERENCES public.workspace_integrations(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: social_login_connections social_login_connection_created_by_id_7ca2ef50_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.social_login_connections
    ADD CONSTRAINT social_login_connection_created_by_id_7ca2ef50_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: social_login_connections social_login_connection_updated_by_id_c13deb42_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.social_login_connections
    ADD CONSTRAINT social_login_connection_updated_by_id_c13deb42_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: social_login_connections social_login_connection_user_id_0e26c0c5_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.social_login_connections
    ADD CONSTRAINT social_login_connection_user_id_0e26c0c5_fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: states state_created_by_id_ff51a50d_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.states
    ADD CONSTRAINT state_created_by_id_ff51a50d_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: states state_project_id_23a65fd6_fk_project_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.states
    ADD CONSTRAINT state_project_id_23a65fd6_fk_project_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: states state_updated_by_id_be298453_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.states
    ADD CONSTRAINT state_updated_by_id_be298453_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: states state_workspace_id_2293282d_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.states
    ADD CONSTRAINT state_workspace_id_2293282d_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: stickies stickies_created_by_id_f72e05c4_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stickies
    ADD CONSTRAINT stickies_created_by_id_f72e05c4_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: stickies stickies_owner_id_6ee3be2b_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stickies
    ADD CONSTRAINT stickies_owner_id_6ee3be2b_fk_users_id FOREIGN KEY (owner_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: stickies stickies_updated_by_id_d660f1fb_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stickies
    ADD CONSTRAINT stickies_updated_by_id_d660f1fb_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: stickies stickies_workspace_id_0094496a_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stickies
    ADD CONSTRAINT stickies_workspace_id_0094496a_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: teams team_created_by_id_725a9101_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT team_created_by_id_725a9101_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: teams team_updated_by_id_79bb36f2_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT team_updated_by_id_79bb36f2_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: teams team_workspace_id_1d56407f_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT team_workspace_id_1d56407f_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_favorites user_favorites_created_by_id_dc025309_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_created_by_id_dc025309_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_favorites user_favorites_parent_id_550512e4_fk_user_favorites_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_parent_id_550512e4_fk_user_favorites_id FOREIGN KEY (parent_id) REFERENCES public.user_favorites(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_favorites user_favorites_project_id_359b527f_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_project_id_359b527f_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_favorites user_favorites_updated_by_id_a1a5ac4a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_updated_by_id_a1a5ac4a_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_favorites user_favorites_user_id_cea7e2d2_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_user_id_cea7e2d2_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_favorites user_favorites_workspace_id_aa90f680_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_favorites
    ADD CONSTRAINT user_favorites_workspace_id_aa90f680_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_notification_preferences user_notification_pr_created_by_id_54dc743a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_preferences
    ADD CONSTRAINT user_notification_pr_created_by_id_54dc743a_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_notification_preferences user_notification_pr_project_id_e0ca17f8_fk_projects_; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_preferences
    ADD CONSTRAINT user_notification_pr_project_id_e0ca17f8_fk_projects_ FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_notification_preferences user_notification_pr_updated_by_id_eb70a86d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_preferences
    ADD CONSTRAINT user_notification_pr_updated_by_id_eb70a86d_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_notification_preferences user_notification_pr_workspace_id_a2321c58_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_preferences
    ADD CONSTRAINT user_notification_pr_workspace_id_a2321c58_fk_workspace FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_notification_preferences user_notification_preferences_user_id_9dccc056_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_notification_preferences
    ADD CONSTRAINT user_notification_preferences_user_id_9dccc056_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_recent_visits user_recent_visits_created_by_id_a655b75f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_recent_visits
    ADD CONSTRAINT user_recent_visits_created_by_id_a655b75f_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_recent_visits user_recent_visits_project_id_e5eecf27_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_recent_visits
    ADD CONSTRAINT user_recent_visits_project_id_e5eecf27_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_recent_visits user_recent_visits_updated_by_id_42b12ef2_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_recent_visits
    ADD CONSTRAINT user_recent_visits_updated_by_id_42b12ef2_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_recent_visits user_recent_visits_user_id_f5153288_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_recent_visits
    ADD CONSTRAINT user_recent_visits_user_id_f5153288_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: user_recent_visits user_recent_visits_workspace_id_362a4e80_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_recent_visits
    ADD CONSTRAINT user_recent_visits_workspace_id_362a4e80_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: users users_avatar_asset_id_50fa2043_fk_file_assets_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_avatar_asset_id_50fa2043_fk_file_assets_id FOREIGN KEY (avatar_asset_id) REFERENCES public.file_assets(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: users users_cover_image_asset_id_b9679cbc_fk_file_assets_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_cover_image_asset_id_b9679cbc_fk_file_assets_id FOREIGN KEY (cover_image_asset_id) REFERENCES public.file_assets(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhook_logs webhook_logs_created_by_id_71e7bc38_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_created_by_id_71e7bc38_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhook_logs webhook_logs_updated_by_id_3d9bad04_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_updated_by_id_3d9bad04_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhook_logs webhook_logs_workspace_id_ffcd0e31_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_workspace_id_ffcd0e31_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhooks webhooks_created_by_id_25aca1b0_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_created_by_id_25aca1b0_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhooks webhooks_updated_by_id_ea35154e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_updated_by_id_ea35154e_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: webhooks webhooks_workspace_id_da5865d7_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_workspace_id_da5865d7_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspaces workspace_created_by_id_10ad894e_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspace_created_by_id_10ad894e_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_home_preferences workspace_home_prefe_workspace_id_b49f76e0_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_home_preferences
    ADD CONSTRAINT workspace_home_prefe_workspace_id_b49f76e0_fk_workspace FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_home_preferences workspace_home_preferences_created_by_id_f31fc163_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_home_preferences
    ADD CONSTRAINT workspace_home_preferences_created_by_id_f31fc163_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_home_preferences workspace_home_preferences_updated_by_id_14ed118a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_home_preferences
    ADD CONSTRAINT workspace_home_preferences_updated_by_id_14ed118a_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_home_preferences workspace_home_preferences_user_id_4087938d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_home_preferences
    ADD CONSTRAINT workspace_home_preferences_user_id_4087938d_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_integrations workspace_integratio_integration_id_6cb0aace_fk_integrati; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integratio_integration_id_6cb0aace_fk_integrati FOREIGN KEY (integration_id) REFERENCES public.integrations(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_integrations workspace_integrations_actor_id_21619aa1_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_actor_id_21619aa1_fk_users_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_integrations workspace_integrations_api_token_id_bdb1759b_fk_api_tokens_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_api_token_id_bdb1759b_fk_api_tokens_id FOREIGN KEY (api_token_id) REFERENCES public.api_tokens(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_integrations workspace_integrations_created_by_id_37639c73_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_created_by_id_37639c73_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_integrations workspace_integrations_updated_by_id_fce01dcb_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_updated_by_id_fce01dcb_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_integrations workspace_integrations_workspace_id_27ebeb6b_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_integrations
    ADD CONSTRAINT workspace_integrations_workspace_id_27ebeb6b_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_members workspace_member_created_by_id_8dc8b040_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_members
    ADD CONSTRAINT workspace_member_created_by_id_8dc8b040_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_member_invites workspace_member_invite_created_by_id_082f21d3_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_member_invites
    ADD CONSTRAINT workspace_member_invite_created_by_id_082f21d3_fk_user_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_member_invites workspace_member_invite_updated_by_id_d31a9c7f_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_member_invites
    ADD CONSTRAINT workspace_member_invite_updated_by_id_d31a9c7f_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_member_invites workspace_member_invite_workspace_id_d935b364_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_member_invites
    ADD CONSTRAINT workspace_member_invite_workspace_id_d935b364_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_members workspace_member_member_id_824f5497_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_members
    ADD CONSTRAINT workspace_member_member_id_824f5497_fk_user_id FOREIGN KEY (member_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_members workspace_member_updated_by_id_1cec0062_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_members
    ADD CONSTRAINT workspace_member_updated_by_id_1cec0062_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_members workspace_member_workspace_id_33f66d4b_fk_workspace_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_members
    ADD CONSTRAINT workspace_member_workspace_id_33f66d4b_fk_workspace_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspaces workspace_owner_id_60a8bafc_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspace_owner_id_60a8bafc_fk_user_id FOREIGN KEY (owner_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_themes workspace_themes_actor_id_0e94172e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_themes
    ADD CONSTRAINT workspace_themes_actor_id_0e94172e_fk_users_id FOREIGN KEY (actor_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_themes workspace_themes_created_by_id_676e2655_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_themes
    ADD CONSTRAINT workspace_themes_created_by_id_676e2655_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_themes workspace_themes_updated_by_id_bba863fe_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_themes
    ADD CONSTRAINT workspace_themes_updated_by_id_bba863fe_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_themes workspace_themes_workspace_id_d1bffad8_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_themes
    ADD CONSTRAINT workspace_themes_workspace_id_d1bffad8_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspaces workspace_updated_by_id_09d249ed_fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspace_updated_by_id_09d249ed_fk_user_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_links workspace_user_links_created_by_id_b9ce7a5d_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_links
    ADD CONSTRAINT workspace_user_links_created_by_id_b9ce7a5d_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_links workspace_user_links_owner_id_37d99444_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_links
    ADD CONSTRAINT workspace_user_links_owner_id_37d99444_fk_users_id FOREIGN KEY (owner_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_links workspace_user_links_project_id_045e0d53_fk_projects_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_links
    ADD CONSTRAINT workspace_user_links_project_id_045e0d53_fk_projects_id FOREIGN KEY (project_id) REFERENCES public.projects(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_links workspace_user_links_updated_by_id_bd0b017f_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_links
    ADD CONSTRAINT workspace_user_links_updated_by_id_bd0b017f_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_links workspace_user_links_workspace_id_1b0a8e22_fk_workspaces_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_links
    ADD CONSTRAINT workspace_user_links_workspace_id_1b0a8e22_fk_workspaces_id FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_preferences workspace_user_prefe_workspace_id_a345adde_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_preferences
    ADD CONSTRAINT workspace_user_prefe_workspace_id_a345adde_fk_workspace FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_preferences workspace_user_preferences_created_by_id_2d566570_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_preferences
    ADD CONSTRAINT workspace_user_preferences_created_by_id_2d566570_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_preferences workspace_user_preferences_updated_by_id_65fed266_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_preferences
    ADD CONSTRAINT workspace_user_preferences_updated_by_id_65fed266_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_preferences workspace_user_preferences_user_id_0ba5007a_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_preferences
    ADD CONSTRAINT workspace_user_preferences_user_id_0ba5007a_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_properties workspace_user_prope_workspace_id_1dc3e2a6_fk_workspace; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_properties
    ADD CONSTRAINT workspace_user_prope_workspace_id_1dc3e2a6_fk_workspace FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_properties workspace_user_properties_created_by_id_6d8d1c4e_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_properties
    ADD CONSTRAINT workspace_user_properties_created_by_id_6d8d1c4e_fk_users_id FOREIGN KEY (created_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_properties workspace_user_properties_updated_by_id_910a2cc5_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_properties
    ADD CONSTRAINT workspace_user_properties_updated_by_id_910a2cc5_fk_users_id FOREIGN KEY (updated_by_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspace_user_properties workspace_user_properties_user_id_b1079e07_fk_users_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace_user_properties
    ADD CONSTRAINT workspace_user_properties_user_id_b1079e07_fk_users_id FOREIGN KEY (user_id) REFERENCES public.users(id) DEFERRABLE INITIALLY DEFERRED;


--
-- Name: workspaces workspaces_logo_asset_id_a784bb00_fk_file_assets_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspaces_logo_asset_id_a784bb00_fk_file_assets_id FOREIGN KEY (logo_asset_id) REFERENCES public.file_assets(id) DEFERRABLE INITIALLY DEFERRED;


--
-- PostgreSQL database dump complete
--


--
-- PostgreSQL database dump
--

-- Dumped from database version 15.7
-- Dumped by pg_dump version 15.7

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

COMMIT;
