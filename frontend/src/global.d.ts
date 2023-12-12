import { Alpine as AlpineType } from 'alpinejs'

declare global {
    var Alpine: AlpineType,

    // Game details
    var uid: number
    var pin: number
    var title: string

    // Websocket protocol definition.
    // Set by server to support both SSL and non-SSL servers.
    var ws_proto: string
}