#!/usr/bin/fish
sudo -iu postgres bash -c 'psql musicdash -ef ~/purge_musicdash_db.sql'
