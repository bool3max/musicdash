**musicdash** is a prototype social media-like platform for dynamically aggregating users' Spotify plays in real time, preserving them in a database and allowing one to derive various interesting statistics about listening habits.

The backend is written primary in Go, and consists of two components that are able to run independently: the primary HTTP REST server (using [Gin](https://gin-gonic.com/)), and the aggregator service which constantly monitors users' accounts for new plays. Both communicate with a PostgreSQL database server.

### Status & roadmap

Currently, the backend is (partially) implemented. Near-future plans include a multi-platform Flutter UI with a focus on listening history viewing and management.