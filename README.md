This is a basic implementation of an http server using golang.

Currently, it supports the following:
- GET requests for a few endpoints -
     -/
     -/echo/{someTextToEcho} - just echoes the text
     -/files/{nameOfFile} -> if u want to fetch the file from server, yes! you can host files on this server too!
- POST requests - /files/{nameOfFile} - creates a file and writes with the request body
- Concurrent connections
- gzip compression for response body

I've implemented this as part of [Build your own HTTP server](https://app.codecrafters.io/courses/http-server/) challenge in [codecrafters](https://app.codecrafters.io/catalog) (a great site for implementing various things like http-server, redis, database and so on from scratch)
