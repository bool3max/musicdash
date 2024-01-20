--
-- PostgreSQL database dump
--

-- Dumped from database version 16.1
-- Dumped by pg_dump version 16.1

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

ALTER TABLE ONLY spotify.track_artist DROP CONSTRAINT track_artist_fk_1;
ALTER TABLE ONLY spotify.track_artist DROP CONSTRAINT track_artist_fk;
ALTER TABLE ONLY spotify.track_album DROP CONSTRAINT track_album_fk_1;
ALTER TABLE ONLY spotify.track_album DROP CONSTRAINT track_album_fk;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_fk1;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_fk;
ALTER TABLE ONLY auth.spotify_token DROP CONSTRAINT spotify_user_fk;
ALTER TABLE ONLY auth.auth_token DROP CONSTRAINT login_session_token_user_fk;
ALTER TABLE ONLY spotify.track DROP CONSTRAINT track_pk;
ALTER TABLE ONLY spotify.track_artist DROP CONSTRAINT track_artist_un;
ALTER TABLE ONLY spotify.track_artist DROP CONSTRAINT track_artist_pk;
ALTER TABLE ONLY spotify.track_album DROP CONSTRAINT track_album_pk;
ALTER TABLE ONLY spotify.images DROP CONSTRAINT images_pk;
ALTER TABLE ONLY spotify.artist DROP CONSTRAINT artist_pk;
ALTER TABLE ONLY spotify.album DROP CONSTRAINT album_pk;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_un;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_pk;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_username_key;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_pwdhash_key;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_pkey;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_email_key;
ALTER TABLE ONLY auth.spotify_token DROP CONSTRAINT spotify_unique;
ALTER TABLE ONLY auth.auth_token DROP CONSTRAINT auth_token_un;
DROP TABLE spotify.track_artist;
DROP TABLE spotify.track_album;
DROP TABLE spotify.track;
DROP TABLE spotify.images;
DROP TABLE spotify.artist;
DROP TABLE spotify.album_artist;
DROP TABLE spotify.album;
DROP TABLE auth."user";
DROP TABLE auth.spotify_token;
DROP TABLE auth.auth_token;
DROP FUNCTION public.generate_uid(size integer);
DROP EXTENSION pgcrypto;
DROP EXTENSION fuzzystrmatch;
DROP EXTENSION citext;
DROP SCHEMA spotify;
DROP SCHEMA auth;
--
-- Name: auth; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA auth;


ALTER SCHEMA auth OWNER TO postgres;

--
-- Name: spotify; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA spotify;


ALTER SCHEMA spotify OWNER TO postgres;

--
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: fuzzystrmatch; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS fuzzystrmatch WITH SCHEMA public;


--
-- Name: EXTENSION fuzzystrmatch; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION fuzzystrmatch IS 'determine similarities and distance between strings';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: generate_uid(integer); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.generate_uid(size integer) RETURNS text
    LANGUAGE plpgsql
    AS $$
DECLARE
  characters TEXT := 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  bytes BYTEA := gen_random_bytes(size);
  l INT := length(characters);
  i INT := 0;
  output TEXT := '';
BEGIN
  WHILE i < size LOOP
    output := output || substr(characters, get_byte(bytes, i) % l + 1, 1);
    i := i + 1;
  END LOOP;
  RETURN output;
END;
$$;


ALTER FUNCTION public.generate_uid(size integer) OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: auth_token; Type: TABLE; Schema: auth; Owner: postgres
--

CREATE TABLE auth.auth_token (
    userid uuid NOT NULL,
    token character varying NOT NULL,
    granted_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


ALTER TABLE auth.auth_token OWNER TO postgres;

--
-- Name: spotify_token; Type: TABLE; Schema: auth; Owner: postgres
--

CREATE TABLE auth.spotify_token (
    userid uuid NOT NULL,
    authtoken character varying NOT NULL,
    refreshtoken character varying NOT NULL,
    expiresat timestamp without time zone NOT NULL
);


ALTER TABLE auth.spotify_token OWNER TO postgres;

--
-- Name: user; Type: TABLE; Schema: auth; Owner: postgres
--

CREATE TABLE auth."user" (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(30) NOT NULL,
    registered_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    email public.citext NOT NULL,
    pwdhash bytea NOT NULL
);


ALTER TABLE auth."user" OWNER TO postgres;

--
-- Name: album; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.album (
    spotifyid character(22) NOT NULL,
    title character varying NOT NULL,
    counttracks integer NOT NULL,
    releasedate date,
    spotifyuri character varying NOT NULL,
    type character varying NOT NULL,
    isrc character varying,
    ean character varying,
    upc character varying
);


ALTER TABLE spotify.album OWNER TO postgres;

--
-- Name: album_artist; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.album_artist (
    spotifyidartist character(22) NOT NULL,
    spotifyidalbum character(22) NOT NULL,
    ismain boolean NOT NULL
);


ALTER TABLE spotify.album_artist OWNER TO postgres;

--
-- Name: artist; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.artist (
    spotifyid character(22) NOT NULL,
    name character varying NOT NULL,
    spotifyuri character varying,
    followers integer
);


ALTER TABLE spotify.artist OWNER TO postgres;

--
-- Name: images; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.images (
    width integer NOT NULL,
    height integer NOT NULL,
    mimetype character varying NOT NULL,
    spotifyid character(22) NOT NULL,
    data bytea NOT NULL,
    url character varying NOT NULL
);


ALTER TABLE spotify.images OWNER TO postgres;

--
-- Name: track; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.track (
    spotifyid character(22) NOT NULL,
    title character varying NOT NULL,
    duration integer NOT NULL,
    tracklistnum integer,
    popularity integer,
    spotifyuri character varying NOT NULL,
    explicit boolean NOT NULL,
    isrc character varying,
    ean character varying,
    upc character varying,
    discnum integer DEFAULT 1 NOT NULL
);


ALTER TABLE spotify.track OWNER TO postgres;

--
-- Name: track_album; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.track_album (
    spotifyidtrack character(22) NOT NULL,
    spotifyidalbum character(22) NOT NULL
);


ALTER TABLE spotify.track_album OWNER TO postgres;

--
-- Name: track_artist; Type: TABLE; Schema: spotify; Owner: postgres
--

CREATE TABLE spotify.track_artist (
    spotifyidtrack character(22) NOT NULL,
    spotifyidartist character(22) NOT NULL,
    ismain boolean NOT NULL
);


ALTER TABLE spotify.track_artist OWNER TO postgres;

--
-- Name: auth_token auth_token_un; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.auth_token
    ADD CONSTRAINT auth_token_un UNIQUE (token);


--
-- Name: spotify_token spotify_unique; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.spotify_token
    ADD CONSTRAINT spotify_unique UNIQUE (userid);


--
-- Name: user user_email_key; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth."user"
    ADD CONSTRAINT user_email_key UNIQUE (email);


--
-- Name: user user_pkey; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- Name: user user_pwdhash_key; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth."user"
    ADD CONSTRAINT user_pwdhash_key UNIQUE (pwdhash);


--
-- Name: user user_username_key; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth."user"
    ADD CONSTRAINT user_username_key UNIQUE (username);


--
-- Name: album_artist album_artist_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.album_artist
    ADD CONSTRAINT album_artist_pk PRIMARY KEY (spotifyidartist, spotifyidalbum);


--
-- Name: album_artist album_artist_un; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.album_artist
    ADD CONSTRAINT album_artist_un UNIQUE (spotifyidalbum, ismain, spotifyidartist);


--
-- Name: album album_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.album
    ADD CONSTRAINT album_pk PRIMARY KEY (spotifyid);


--
-- Name: artist artist_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.artist
    ADD CONSTRAINT artist_pk PRIMARY KEY (spotifyid);


--
-- Name: images images_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.images
    ADD CONSTRAINT images_pk PRIMARY KEY (url, spotifyid, width, height);


--
-- Name: track_album track_album_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_album
    ADD CONSTRAINT track_album_pk PRIMARY KEY (spotifyidalbum, spotifyidtrack);


--
-- Name: track_artist track_artist_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_artist
    ADD CONSTRAINT track_artist_pk PRIMARY KEY (spotifyidtrack, spotifyidartist);


--
-- Name: track_artist track_artist_un; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_artist
    ADD CONSTRAINT track_artist_un UNIQUE (spotifyidtrack, spotifyidartist, ismain);


--
-- Name: track track_pk; Type: CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track
    ADD CONSTRAINT track_pk PRIMARY KEY (spotifyid);


--
-- Name: auth_token login_session_token_user_fk; Type: FK CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.auth_token
    ADD CONSTRAINT login_session_token_user_fk FOREIGN KEY (userid) REFERENCES auth."user"(id);


--
-- Name: spotify_token spotify_user_fk; Type: FK CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.spotify_token
    ADD CONSTRAINT spotify_user_fk FOREIGN KEY (userid) REFERENCES auth."user"(id);


--
-- Name: album_artist album_artist_fk; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.album_artist
    ADD CONSTRAINT album_artist_fk FOREIGN KEY (spotifyidartist) REFERENCES spotify.artist(spotifyid);


--
-- Name: album_artist album_artist_fk1; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.album_artist
    ADD CONSTRAINT album_artist_fk1 FOREIGN KEY (spotifyidalbum) REFERENCES spotify.album(spotifyid);


--
-- Name: track_album track_album_fk; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_album
    ADD CONSTRAINT track_album_fk FOREIGN KEY (spotifyidtrack) REFERENCES spotify.track(spotifyid);


--
-- Name: track_album track_album_fk_1; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_album
    ADD CONSTRAINT track_album_fk_1 FOREIGN KEY (spotifyidalbum) REFERENCES spotify.album(spotifyid);


--
-- Name: track_artist track_artist_fk; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_artist
    ADD CONSTRAINT track_artist_fk FOREIGN KEY (spotifyidtrack) REFERENCES spotify.track(spotifyid);


--
-- Name: track_artist track_artist_fk_1; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track_artist
    ADD CONSTRAINT track_artist_fk_1 FOREIGN KEY (spotifyidartist) REFERENCES spotify.artist(spotifyid);


--
-- PostgreSQL database dump complete
