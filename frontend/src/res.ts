export const gong = new Audio("/static/music/gong.mp3");
export const qmusic_long = new Audio("/static/music/decisions.mp3");
export const qmusic = new Audio("/static/music/nerves-1.mp3");
export const qmusic_short = new Audio("/static/music/nerves-2.mp3");
export const qmusic_vshort = new Audio("/static/music/nerves-3.mp3");

export function preload_media() {
    gong.load();
    qmusic_long.load();
    qmusic.load();
    qmusic_short.load()
    qmusic_vshort.load();
}

/*
 * stop_media calls pause on a player and seeks back to the beginning.
 */
export function stop_media(media: HTMLAudioElement): void {
    media.pause();
    media.currentTime = 0;
}