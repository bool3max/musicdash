#!/usr/bin/fish

# dump script to postgres user home dir
sudo -u postgres pg_dump -sc -f '/var/lib/postgres/data/musicdash.sql' musicdash

sudo mv /var/lib/postgres/data/musicdash.sql $argv[1]
