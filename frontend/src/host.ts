/*
 *  Gahoot! A self-hostable, minimal rewrite of Kahoot! in Go
 *  Copyright 2022 - Ethan Marshall
 *
 *  Gameplay scripts
 */

import * as common from "./common"
import Alpine from "alpinejs"

// Page lifetime variables
let conn: WebSocket
let host: HostState

// Set up alpine on the window
// For debugging purposes
window.Alpine = Alpine

enum States {
    JoinWaiting = 1,
    StartCountdown,
    QuestionCountdown,
    QuestionAsk,
    QuestionAnswer,
    GameOver
}

interface PlayerData {
    id: number
    name: string
    score: number
    correct: number
}

interface QuestionData {
    title: string
    image?: string
    time: number
    answers: {
        title: string
        correct: boolean
    }[]

    index: number
    total: number
}

interface Player extends PlayerData {
    connected: boolean
    loading: boolean
}

// PlayerState is the current datamodel for the client.
//
// Any code which mutates the state of the application based
// on events from the server or anything else *must* be a member
// of this class for the changes to be reflected in the DOM.
class HostState {
    pin: number
    title: string

    private state: common.GameState<HostState>
    stateID: number

    connected: boolean

    countdownTitle: string
    countdownFull: boolean
    countdownCount: number
    private countdownHndl: number

    players: Player[]
    startError: boolean

    question: QuestionData
    gotAnswers: number
    questionCountdown: number
    private questionCountdownHndl: number

    // Initializes data defaults
    //
    // NOTE: Does not do any interaction with events!
    // All event handlers must be hooked in init()
    constructor(game: number, title: string) {
        this.connected = false
        this.pin = game
        this.title = title
        this.players = []
        this.startError = false

        this.countdownTitle = ""
        this.countdownFull = false
        this.countdownCount = 10
        this.countdownHndl = 0

        this.question = {
            title: "Error!",
            image: "https://developer.valvesoftware.com/w/images/5/5b/Missing_textures_example.png",
            time: 10,
            answers: [],
            index: 1,
            total: 10,
        }
        this.gotAnswers = 0
        this.questionCountdown = this.question.time
        this.questionCountdownHndl = 0

        this.state = this.stateWaitingJoin
        this.stateID = States.JoinWaiting
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
        common.SendMessage(conn, "host", this.pin)
        console.log("now hosting game " + this.pin.toString())
        this.handleConnection(true)
    }

    // handleConnection is called when a websocket connection changes state
    handleConnection(connected: boolean) {
        this.connected = connected

        if (!this.connected && this.stateID != States.GameOver) {
            document.location.href = "/create/"
        }
    }

    // handleMsg is called when a websocket message arrives
    //
    // This must *only* be used to do parsing and state shifts
    // and must never mutate state itself
    handleMsg(ev: MessageEvent) {
        let [action, ...rest]: string[] = ev.data.toString().split(" ")
        let msg: common.GameMessage = {
            action: action,
            data: JSON.parse(rest.join(" "))
        }

        this.state = this.state(msg)
    }

    // STATE FUNCTIONS
    // ---------------

    // waiting on more joining, or a message signalling join ends
    stateWaitingJoin(ev: common.GameMessage): common.GameState<HostState> {
        this.stateID = States.JoinWaiting


        let plr = <PlayerData>ev.data
        switch (ev.action) {
            // Remove player
            case "rmplr":
                console.log("removed player "+plr.name)
            this.players = this.players.filter(pl => pl.name != plr.name)
            return this.state
            case "dcplr":
                // For now, also just remove the player
                this.players.map(pl => {
                if (pl.name == plr.name) {
                    pl.connected = false
                }
            })
            return this.state
        }
        if (ev.action != "plr") {
            console.warn("unexpected message: "+ev.action)
            return this.state
        }

        for (var i: number = 0; i < this.players.length; i++) {
            if (this.players[i].id == plr.id) {
                if (!this.players[i].connected) {
                    this.players[i].connected = true
                    return this.state
                }
                console.warn("duplicate player join notification received!")
                return this.state
            }
        }

        this.players.push({
            id: plr.id,
            name: plr.name,
            score: plr.score,
            correct: plr.correct,

            connected: true,
            loading: false,
        })
        console.log(plr.name+" (ID: "+plr.id.toString()+") joined")
        return this.state
    }

    // message received signalling countdown ended
    stateStartCountdown(ev: common.GameMessage): common.GameState<HostState> {
        if (ev.action != "sack") {
            console.warn("unexpected message: "+ev.action)
            return this.state
        }

        return this.stateQuestionCountdown
    }

    // question countdown begins
    stateQuestionCountdown(ev: common.GameMessage): common.GameState<HostState> {
        if (ev.action != "ques") {
            console.warn("expected question, got "+ev.action)
            return this.state
        }

        this.stateID = States.QuestionCountdown
        this.question = <QuestionData>ev.data
        this.startCountdown(5, {
            action: "sans",
            data: {},
        }, this.question.title)
        return this.stateQuestion
    }

    // question countdown ended; waiting for submissions
    stateQuestion(ev: common.GameMessage): common.GameState<HostState> {
        switch (ev.action) {
            case "quack":
                this.stateID = States.QuestionAsk
                this.questionCountdown = this.question.time
                this.questionCountdownHndl = window.setInterval(() => {
                    this.questionCountdown--;
                    if (this.questionCountdown <= 0) {
                        this.skip()
                        return
                    }
                }, 1000)
                return this.state
            case "nans":
                this.gotAnswers++;
                return this.state
            case "qend":
                this.stateID = States.QuestionAnswer
                return this.stateFeedback
            default:
                console.warn("expected question acknowledge, got "+ev.action)
                return this.state
        }
    }

    stateFeedback(ev: common.GameMessage): common.GameState<HostState> {
        return this.state
    }

    // FRONTEND FUNCTIONS
    // ------------------

    // Send the primary start message and schedule the secondary
    //
    // Tells the game server to start countdowns on other client screens,
    // as well as waiting for the next instruction. Schedules the
    // secondary start message, which instantly shifts us to the first
    // question's countdown.
    startGame(): void {
        // Minimum three players require before starting
        if (this.players.length < 3) {
            this.startError = true
            setTimeout(() => {this.startError = false}, 1000)
            return
        }

        this.stateID = States.StartCountdown
        common.SendMessage(conn, "count", 5)
        let startmsg = <common.GameMessage>{
            action: "start",
            data: {}
        }
        this.startCountdown(5, startmsg, this.title)
        this.state = this.stateStartCountdown
    }

    // Ends the current question
    //
    // Desipte the name, this does not have to be a manual skip. The time
    // running out also calls this function.
    //
    // Does not mutate state, as we have to wait for a "qend" packet from the
    // server first.
    skip(): void {
        window.clearInterval(this.questionCountdownHndl)
        common.SendMessage(conn, "time", {})
    }

    // Start the visual countdown on screen.
    // If title is provided, the countdown is a "full" countdown, showing
    // an image etc.
    startCountdown(length: number, msg: common.GameMessage, title?: string) {
        if (length == 0) {
            return this.state
        }
        if (title) {
            this.countdownFull = true
            this.countdownTitle = title
        }
        this.countdownCount = length

        this.countdownHndl = window.setInterval(() => {
            this.countdownCount--;
            if (this.countdownCount <= 0) {
                common.SendMessage(conn, msg.action, msg.data)
                clearInterval(this.countdownHndl)
            }
        }, 1000)
    }

    // Request the server to kick a player
    kickPlayer(id: number): void {
        this.players.map(pl => {
            if (pl.id == id) {
                pl.loading = true
            }
        })
        common.SendMessage(conn, "kick", id)
    }
}

// Main frontend init code
document.addEventListener("DOMContentLoaded", () => {
    console.log("Gahoot! host scripts loaded")
    console.log("Joining game " + window.pin + " as the host")

    // Init our global objects
    host = new HostState(window.pin, window.title)

    // Load information
    let url = common.HostEndpoint + window.pin.toString()
    conn = new WebSocket(url)

    // Start tracking our data using Alpine
    Alpine.store("host", host)
    Alpine.start()
})
