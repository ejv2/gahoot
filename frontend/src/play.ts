/*
 *  Gahoot! A self-hostable, minimal rewrite of Kahoot! in Go
 *  Copyright 2022 - Ethan Marshall
 *
 *  Gameplay scripts
 */

import Alpine from "alpinejs"

// Gameplay constants
const PlayEndpoint = "ws://" + location.host + "/api/play/"

// Page lifetime variables
let conn: WebSocket
let plr: PlayerState

// GameMessage represents a message received over the websocket channel
interface GameMessage {
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
interface GameState {
    (this: PlayerState, ev: GameMessage): GameState
}

// Set up alpine on the window
// For debugging purposes
window.Alpine = Alpine

// PlayerState is the current datamodel for the client.
//
// Any code which mutates the state of the application based
// on events from the server or anything else *must* be a member
// of this class for the changes to be reflected in the DOM.
class PlayerState {
    private pin: number
    private uid: number

    private state: GameState

    connected: boolean
    points: number
    rank: number

    // Initializes data defaults
    //
    // NOTE: Does not do any interaction with events!
    // All event handlers must be hooked in init()
    constructor(game: number, user: number) {
        this.connected = false
        this.points = this.rank = 0

        this.pin = game
        this.uid = user

        this.state = function e(ev: GameMessage): GameState {
                console.log(ev)
                return this.state
        }
    }

    // Initializes event listeners such that Alpine will track changes for us
    //
    // Data should not be mutated in here, unless the mutation is *really* simple
    // Delegate to methods where appropriate
    init() {
        conn.onopen = () => {this.initConn()}
        conn.onclose = (ev: CloseEvent) => {this.handleConnection(false);console.log(ev)}
        conn.onmessage = (e: MessageEvent) => {this.handleMsg(e)}
        conn.onerror = () => {this.handleConnection(false)}
    }

    // Initializes the connection and internal state by sending the ident
    // packets
    initConn() {
        sendMessage(conn, "ident", this.uid)
        console.log("authenticated to game " + this.pin.toString())
        this.handleConnection(true)
    }

    // handleConnection is called when a websocket connection changes state
    handleConnection(connected: boolean) {
        this.connected = connected
    }

    // handleMsg is called when a websocket message arrives
    //
    // This must *only* be used to do parsing and state shifts
    // and must never mutate state itself
    handleMsg(ev: MessageEvent) {
        let [action, ...rest]: string[] = ev.data.toString().split(" ")
        let msg: GameMessage = {
            action: action,
            data: JSON.parse(rest.join(" "))
        }

        this.state = this.state(msg)
    }
}

function sendMessage(ws: WebSocket, action: string, body: any) {
    ws.send(action + " " + body.toString())
}

// Main frontend init code
document.addEventListener("DOMContentLoaded", () => {
    console.log("Gahoot! client scripts loaded")
    console.log("Joining game " + window.pin + " as " + window.uid)

    // Init our global objects
    plr = new PlayerState(window.pin, window.uid)

    // Load information
    let url = PlayEndpoint + window.pin.toString()
    conn = new WebSocket(url)

    // Start tracking our data using Alpine
    Alpine.store("game", plr)
    Alpine.start()
})
