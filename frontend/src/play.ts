/*
 *  Gahoot! A self-hostable, minimal rewrite of Kahoot! in Go
 *  Copyright 2022 - Ethan Marshall
 *
 *  Gameplay scripts
 */

import Alpine from "alpinejs"

// Gameplay constants
export const PlayEndpoint = "ws://" + location.host + "/api/play/"

// Page lifetime variables
let conn: WebSocket
let plr: PlayerState

// Set up alpine on the window
// For debugging purposes
window.Alpine = Alpine

// PlayerState represents the current state of the page for the player.
// It holds and manages the updating of dynamic data to be displayed by
// the HTML engine.
class PlayerState {

    connected: boolean
    points: number
    rank: number

    // Initializes data defaults
    //
    // NOTE: Does not do any interaction with events!
    // All event handlers must be hooked in init()
    constructor() {
        this.connected = false
        this.points = this.rank = 0
    }

    // Initializes event listeners such that Alpine will track changes for us
    //
    // Data should not be mutated in here, if possible
    // Delegate to methods where appropriate
    init() {
        conn.onopen = () => {this.handleConnection(true)}
        conn.onclose = () => {this.handleConnection(false)}
        conn.onmessage = (e: MessageEvent) => {this.handleMsg(e)}
        conn.onerror = () => {this.handleConnection(false)}
    }

    // handleConnection is called when a websocket connection changes state
    handleConnection(conn: boolean) {
        this.connected = conn
        // TODO: Add more handling here
    }

    // handleMsg is called when a websocket message arrives
    //
    // Parsing should take place here, but actual mutation *must*
    // take place in separate methods.
    handleMsg(ev: MessageEvent) {
        console.log(ev)
    }
}

// Main frontend init code
document.addEventListener("DOMContentLoaded", () => {
    console.log("Gahoot! client scripts loaded")
    console.log("Joining game " + window.pin + " as " + window.uid)

    // Init our global objects
    plr = new PlayerState()

    // Load information
    let url = PlayEndpoint + window.pin.toString()
    conn = new WebSocket(url)

    // Start tracking our data using Alpine
    Alpine.store("game", plr)
    Alpine.start()
})
