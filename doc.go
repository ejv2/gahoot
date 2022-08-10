/*
Gahoot is a lightweight, distributed, FOSS rewrite of Gahoot in Go. It has the
following design goals:
 1. Be lightweight enough to be run by individual users
 2. To be cross platform, both the server and client
 3. To encourage a distributed network of game servers and players

This IS NOT a feature complete remake of Kahoot. Instead, it is an opinionated
rewrite of what I think makes the game special. It IS NOT the same game and IS
NOT intended as a hard replacement. It IS intended to be fun and useful,
however.

Gahoot can be run by simply running the binary "./gahoot". If this causes an
error to do with "missing frontend", make sure you pulled from SCM properly
and/or have all included distribution files; you need everything included.

Configuration is stored in "config.gahoot" in the same directory as program
startup. We do not crawl your filesystem. No files outside the application
directory (the value of getcwd() at startup) should be accessed or need to be
accessible to the program.

Quizzes stored on disk are, by default, stored in a directory called "quizzes"
in the application directory. This is crawled once at startup and is never
accessed again. Quizzes can be loaded into memory via the web interface by
anybody at any time, although Gahoot manages this carefully to avoid OOM
or DOS-based attacks on the server. Quizzes are stored as JSON-encoded dumps of
all quiz metadata and all questions provided.
*/
package main
