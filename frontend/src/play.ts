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
let plr: PlayerState

// Possible game state IDs
enum States {
    Loading = 1,
    Waiting,
    Countdown,
    Question,
    Answer,
    Finished
}

interface CountdownData {
    length: number,
    title: string,
}

interface QuestionData {
    title: string,
    image: string,
    answers: string[]
}

interface FeedbackData {
    correct: boolean,
    points: number,
    placement: number,
    behind: string,
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

    private state: common.GameState<PlayerState>
    stateID: States

    connected: boolean
    points: number
    rank: number

    countdown: CountdownData
    countdownCount: number
    private countdownHndl: number

    question: QuestionData
    feedback: FeedbackData
    submitSpinner: boolean

    // Initializes data defaults
    //
    // NOTE: Does not do any interaction with events!
    // All event handlers must be hooked in init()
    constructor(game: number, user: number) {
        this.connected = false
        this.points = this.rank = 0

        this.pin = game
        this.uid = user

        this.countdown = {
            title: "",
            length: 5,
        }
        this.countdownCount = this.countdown.length
        this.countdownHndl = 0

        this.question = {
            title: "Error!",
            image: "https://developer.valvesoftware.com/w/images/5/5b/Missing_textures_example.png",
            answers: [
                "1",
                "2",
                "3",
                "4"
            ],
        }
        this.feedback = {
            correct: false,
            points: 0,
            behind: "Gabe Newell",
            placement: 3
        }
        this.submitSpinner = false

        this.state = this.stateWaiting
        this.stateID = States.Loading
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
        setTimeout(() => {
            common.SendMessage(conn, "ident", this.uid)
            console.log("authenticated to game " + this.pin.toString())
            this.handleConnection(true)
            this.stateID = States.Waiting
        }, 700)
    }

    // handleConnection is called when a websocket connection changes state
    handleConnection(connected: boolean) {
        this.connected = connected

        if (!this.connected && this.stateID != States.Finished) {
            window.location.href = "/join";
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

    startCountdown(ev: common.GameMessage) {
        this.countdown = <CountdownData>ev.data;
        if (this.countdown.length == 0) {
            return this.state
        }

        this.countdownHndl = window.setInterval(() => {
            this.countdownCount--
            if (this.countdownCount == 0) {
                clearInterval(this.countdownHndl)
                common.SendMessage(conn, "ack", {})

                this.state = this.stateQuestion
            }
        }, 1000)
    }

    // STATE FUNCTIONS
    // ---------------

    // Waiting for game to start
    stateWaiting(ev: common.GameMessage): common.GameState<PlayerState> {
        if (ev.action != "starting") {
            console.warn("Expected starting message, got " + ev.action)
            return this.stateWaiting
        }

        this.stateID = States.Countdown
        this.startCountdown(ev)

        return this.stateQuestion
    }

    // Show the question on screen
    stateQuestion(ev: common.GameMessage): common.GameState<PlayerState> {
        if (ev.action != "ques") {
            console.warn("Expecting question, got " + ev.action)
            return this.state
        }

        this.stateID = States.Question
        this.question = <QuestionData>ev.data
        return this.stateFeedback
    }

    // "Classroom genius?"
    stateWaitingFeedback(ev: common.GameMessage): common.GameState<PlayerState> {
        if (ev.action != "ansack") {
            console.warn("Expecting answer acknowledge, got " + ev.action)
            return this.state
        }

        return this.stateFeedback
    }

    // Showing if answer was correct or not
    stateFeedback(ev: common.GameMessage): common.GameState<PlayerState> {
        this.feedback = <FeedbackData>ev.data

        this.points += this.feedback.points
        this.rank = this.feedback.placement

        switch (ev.action) {
        case "quend":
            break
        case "res":
            this.stateID = States.Finished
            return this.stateEnding
        default:
            console.warn("Expecting results/feedback, got " + ev.action)
            return this.state
        }

        this.stateID = States.Answer

        this.startCountdown(ev)
        return this.stateQuestion
    }

    // Put the FSM into an infinite loop - we are done
    // Any further messages are ignored, including closes
    stateEnding(ev: common.GameMessage): common.GameState<PlayerState> {
        return this.stateEnding
    }
}

// Main frontend init code
document.addEventListener("DOMContentLoaded", () => {
    console.log("Gahoot! client scripts loaded")
    console.log("Joining game " + window.pin + " as " + window.uid)

    // Init our global objects
    plr = new PlayerState(window.pin, window.uid)

    // Load information
    let url = common.PlayEndpoint + window.pin.toString()
    conn = new WebSocket(url)

    // Start tracking our data using Alpine
    Alpine.store("game", plr)
    Alpine.start()
})
