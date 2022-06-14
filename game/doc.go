// Package game implements the main game controller and contains
// representations for a current game state. Every server instance is an
// instance of a single game coordinator, which is responsible for multiplexing
// over multiple different game instances with associated game state. Each game
// contains at least two players, each of which has an associated websocket
// connection.
//
// To achieve this efficiently, a tiered flow of data is used. The
// GameCoordinator is a singleton and owns all game instances for a particular
// game server. Each game coordinator will always be handling zero or more
// games at once. Each game further contains at least two player instances at a
// time. Players are owned by a Game and handled by that instance alone.
package game
