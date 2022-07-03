/*
 *  Gahoot! A self-hostable, minimal rewrite of Kahoot! in Go
 *  Copyright 2022 - Ethan Marshall
 *
 *  Gameplay scripts
 */

// Endpoint locations
export const PlayEndpoint = "ws://" + location.host + "/api/play/"
export const HostEndpoint = "ws://" + location.host + "/api/host/"


// GameMessage represents a message received over the websocket channel
export interface GameMessage {
    action: string
    data: any
}

// State interface
// Represents a single state in the client finite state machine
//
// Each time a message is received, the state is called and returns a new
// state, which could be itself to remain in this state.
//
// This design is inspired by Rob Pike's 2010 talk Lexical Scanning in Go:
// https://youtu.be/HxaD_trXwRE?t=740
export interface GameState<T> {
    (this: T, ev: GameMessage): GameState<T>
}

// Sends a properly formatted message over ws
export function SendMessage(ws: WebSocket, action: string, body: any) {
    ws.send(action + " " + JSON.stringify(body))
}
