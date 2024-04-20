#!/usr/bin/fish

# generate SQL script with pg_dump as postgres linux user to home dir
#   -s : only dump schemas and not the data that the tables contain
#   -c : Output commands to DROP all the dumped database objects prior to
#        outputting the commands for creating them. This is useful because I am
#        constantly developing on two separate machines and SQL commands to create
#        schemas fail if they already exist.

sudo -u postgres pg_dump -sc -f '/var/lib/postgres/data/musicdash.sql' musicdash

# move temp file from postgres user home dir to specified dir
sudo mv /var/lib/postgres/data/musicdash.sql $argv[1]
