# Wuziqi

Simple multiplayer implementation of the game [Wuziqi/Gomoku](https://en.wikipedia.org/wiki/Gomoku) using vanilla JS and a websocket. No logins, games can be played together via link sharing.
 Game states are stored in memory only and forgotten once the game is over.

Configuration works via the command line options. Hosting the game in a subdirectory is supported by passing the `X-Forwarded-Prefix` header in the reverse proxy.

A nix flake is provided for convenient hosting on a nixos server.
