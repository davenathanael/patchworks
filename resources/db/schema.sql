\restrict dbmate

-- Dumped from database version 18.3
-- Dumped by pg_dump version 18.3 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: moddatetime; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS moddatetime WITH SCHEMA public;


--
-- Name: EXTENSION moddatetime; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION moddatetime IS 'functions for tracking last modification time';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: bookmark_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bookmark_tags (
    bookmark_id uuid NOT NULL,
    tag text NOT NULL,
    author_id uuid NOT NULL
);


--
-- Name: bookmarks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.bookmarks (
    id uuid NOT NULL,
    url text NOT NULL,
    title text NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    archived_at timestamp without time zone,
    author_id uuid
);


--
-- Name: collection_bookmarks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.collection_bookmarks (
    collection_id uuid NOT NULL,
    bookmark_id uuid NOT NULL,
    added_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: collection_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.collection_members (
    collection_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role text NOT NULL,
    added_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: collections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.collections (
    id uuid NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    id uuid NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp without time zone NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid NOT NULL,
    email text NOT NULL,
    password_hash text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_login_at timestamp without time zone
);


--
-- Name: bookmark_tags bookmark_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bookmark_tags
    ADD CONSTRAINT bookmark_tags_pkey PRIMARY KEY (bookmark_id, tag, author_id);


--
-- Name: bookmarks bookmarks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bookmarks
    ADD CONSTRAINT bookmarks_pkey PRIMARY KEY (id);


--
-- Name: collection_bookmarks collection_bookmarks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collection_bookmarks
    ADD CONSTRAINT collection_bookmarks_pkey PRIMARY KEY (collection_id, bookmark_id);


--
-- Name: collection_members collection_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collection_members
    ADD CONSTRAINT collection_members_pkey PRIMARY KEY (collection_id, user_id);


--
-- Name: collections collections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collections
    ADD CONSTRAINT collections_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);




--
-- Name: bookmarks update_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_updated_at BEFORE UPDATE ON public.bookmarks FOR EACH ROW EXECUTE FUNCTION public.moddatetime('updated_at');


--
-- Name: collections update_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_updated_at BEFORE UPDATE ON public.collections FOR EACH ROW EXECUTE FUNCTION public.moddatetime('updated_at');


--
-- Name: users update_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER update_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.moddatetime('updated_at');


--
-- Name: bookmark_tags bookmark_tags_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bookmark_tags
    ADD CONSTRAINT bookmark_tags_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: bookmark_tags bookmark_tags_bookmark_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bookmark_tags
    ADD CONSTRAINT bookmark_tags_bookmark_id_fkey FOREIGN KEY (bookmark_id) REFERENCES public.bookmarks(id) ON DELETE CASCADE;


--
-- Name: bookmarks bookmarks_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.bookmarks
    ADD CONSTRAINT bookmarks_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: collection_bookmarks collection_bookmarks_bookmark_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collection_bookmarks
    ADD CONSTRAINT collection_bookmarks_bookmark_id_fkey FOREIGN KEY (bookmark_id) REFERENCES public.bookmarks(id) ON DELETE CASCADE;


--
-- Name: collection_bookmarks collection_bookmarks_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collection_bookmarks
    ADD CONSTRAINT collection_bookmarks_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.collections(id) ON DELETE CASCADE;


--
-- Name: collection_members collection_members_collection_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collection_members
    ADD CONSTRAINT collection_members_collection_id_fkey FOREIGN KEY (collection_id) REFERENCES public.collections(id) ON DELETE CASCADE;


--
-- Name: collection_members collection_members_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.collection_members
    ADD CONSTRAINT collection_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: sessions sessions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict dbmate


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20260428152636'),
    ('20260501120000'),
    ('20260510095913');
