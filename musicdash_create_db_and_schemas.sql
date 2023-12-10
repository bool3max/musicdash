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

--
-- Name: musicdash; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE musicdash WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.UTF-8';


ALTER DATABASE musicdash OWNER TO postgres;

\connect musicdash

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
-- Name: spotify; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA spotify;


ALTER SCHEMA spotify OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

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
    upc character varying
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
--

