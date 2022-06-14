import { Alpine as AlpineType } from 'alpinejs'

declare global {
    var Alpine: AlpineType,

    // Game details
    var uid: number
    var pin: number
}
