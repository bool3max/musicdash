--
-- PostgreSQL database dump
--

-- Dumped from database version 16.2
-- Dumped by pg_dump version 16.2

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
ALTER TABLE ONLY spotify.track DROP CONSTRAINT track_album_fk;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_fk1;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_fk;
ALTER TABLE ONLY public.plays DROP CONSTRAINT plays_user_fk;
ALTER TABLE ONLY auth.user_spotify DROP CONSTRAINT user_spotify_user_fk;
ALTER TABLE ONLY auth.user_profile_img DROP CONSTRAINT user_profile_img_user_fk;
ALTER TABLE ONLY auth.spotify_token DROP CONSTRAINT spotify_user_fk;
ALTER TABLE ONLY auth.auth_token DROP CONSTRAINT login_session_token_user_fk;
ALTER TABLE ONLY spotify.track DROP CONSTRAINT track_pk;
ALTER TABLE ONLY spotify.track_artist DROP CONSTRAINT track_artist_un;
ALTER TABLE ONLY spotify.track_artist DROP CONSTRAINT track_artist_pk;
ALTER TABLE ONLY spotify.images DROP CONSTRAINT images_pk;
ALTER TABLE ONLY spotify.artist DROP CONSTRAINT artist_pk;
ALTER TABLE ONLY spotify.album DROP CONSTRAINT album_pk;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_un;
ALTER TABLE ONLY spotify.album_artist DROP CONSTRAINT album_artist_pk;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_username_key;
ALTER TABLE ONLY auth.user_spotify DROP CONSTRAINT user_spotify_unique_1;
ALTER TABLE ONLY auth.user_spotify DROP CONSTRAINT user_spotify_unique;
ALTER TABLE ONLY auth.user_spotify DROP CONSTRAINT user_spotify_pk;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_pwdhash_key;
ALTER TABLE ONLY auth.user_profile_img DROP CONSTRAINT user_profile_img_pk;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_pkey;
ALTER TABLE ONLY auth."user" DROP CONSTRAINT user_email_key;
ALTER TABLE ONLY auth.spotify_token DROP CONSTRAINT spotify_token_pk;
ALTER TABLE ONLY auth.auth_token DROP CONSTRAINT auth_token_un;
DROP TABLE spotify.track_artist;
DROP TABLE spotify.track;
DROP TABLE spotify.images;
DROP TABLE spotify.artist;
DROP TABLE spotify.album_artist;
DROP TABLE spotify.album;
DROP TABLE public.plays;
DROP TABLE auth.user_spotify;
DROP TABLE auth.user_profile_img;
DROP TABLE auth."user";
DROP TABLE auth.spotify_token;
DROP TABLE auth.auth_token;
DROP FUNCTION public.generate_uid(size integer);
DROP EXTENSION pgcrypto;
DROP EXTENSION fuzzystrmatch;
DROP EXTENSION citext;
DROP SCHEMA spotify;
-- *not* dropping schema, since initdb creates it
DROP SCHEMA auth;
--
-- Name: auth; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA auth;


ALTER SCHEMA auth OWNER TO postgres;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: postgres
--

-- *not* creating schema, since initdb creates it


ALTER SCHEMA public OWNER TO postgres;

--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON SCHEMA public IS '';


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
    accesstoken character varying NOT NULL,
    refreshtoken character varying NOT NULL,
    expiresat timestamp with time zone NOT NULL
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
    pwdhash bytea,
    refreshedat timestamp with time zone
);


ALTER TABLE auth."user" OWNER TO postgres;

--
-- Name: user_profile_img; Type: TABLE; Schema: auth; Owner: postgres
--

CREATE TABLE auth.user_profile_img (
    userid uuid NOT NULL,
    width integer NOT NULL,
    height integer NOT NULL,
    data bytea NOT NULL,
    uploaded_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    size integer NOT NULL
);


ALTER TABLE auth.user_profile_img OWNER TO postgres;

--
-- Name: user_spotify; Type: TABLE; Schema: auth; Owner: postgres
--

CREATE TABLE auth.user_spotify (
    userid uuid NOT NULL,
    spotify_displayname character varying NOT NULL,
    spotify_followers integer,
    spotify_uri character varying NOT NULL,
    profile_image_url character varying NOT NULL,
    profile_image_width integer NOT NULL,
    profile_image_height integer NOT NULL,
    country character varying,
    spotify_email character varying NOT NULL,
    spotify_url character varying NOT NULL,
    spotify_id character varying NOT NULL
);


ALTER TABLE auth.user_spotify OWNER TO postgres;

--
-- Name: plays; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.plays (
    userid uuid NOT NULL,
    spotifyid character varying NOT NULL,
    at timestamp with time zone NOT NULL
);


ALTER TABLE public.plays OWNER TO postgres;

--
-- Name: TABLE plays; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.plays IS 'Stores all individual plays by musicdash users. One recorded play per row.';


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
    discnum integer DEFAULT 1 NOT NULL,
    spotifyidalbum character varying
);


ALTER TABLE spotify.track OWNER TO postgres;

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
-- Name: spotify_token spotify_token_pk; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.spotify_token
    ADD CONSTRAINT spotify_token_pk PRIMARY KEY (userid);


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
-- Name: user_profile_img user_profile_img_pk; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.user_profile_img
    ADD CONSTRAINT user_profile_img_pk PRIMARY KEY (userid);


--
-- Name: user user_pwdhash_key; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth."user"
    ADD CONSTRAINT user_pwdhash_key UNIQUE (pwdhash);


--
-- Name: user_spotify user_spotify_pk; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.user_spotify
    ADD CONSTRAINT user_spotify_pk PRIMARY KEY (userid, spotify_id);


--
-- Name: user_spotify user_spotify_unique; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.user_spotify
    ADD CONSTRAINT user_spotify_unique UNIQUE (userid);


--
-- Name: user_spotify user_spotify_unique_1; Type: CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.user_spotify
    ADD CONSTRAINT user_spotify_unique_1 UNIQUE (spotify_id);


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
-- Name: user_profile_img user_profile_img_user_fk; Type: FK CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.user_profile_img
    ADD CONSTRAINT user_profile_img_user_fk FOREIGN KEY (userid) REFERENCES auth."user"(id);


--
-- Name: user_spotify user_spotify_user_fk; Type: FK CONSTRAINT; Schema: auth; Owner: postgres
--

ALTER TABLE ONLY auth.user_spotify
    ADD CONSTRAINT user_spotify_user_fk FOREIGN KEY (userid) REFERENCES auth."user"(id);


--
-- Name: plays plays_user_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.plays
    ADD CONSTRAINT plays_user_fk FOREIGN KEY (userid) REFERENCES auth."user"(id);


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
-- Name: track track_album_fk; Type: FK CONSTRAINT; Schema: spotify; Owner: postgres
--

ALTER TABLE ONLY spotify.track
    ADD CONSTRAINT track_album_fk FOREIGN KEY (spotifyidalbum) REFERENCES spotify.album(spotifyid);


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
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;


--
-- PostgreSQL database dump complete
--

