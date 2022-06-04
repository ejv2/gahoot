// Snowpack Configuration File
// See all supported options: https://www.snowpack.dev/reference/configuration

/** @type {import("snowpack").SnowpackUserConfig } */
module.exports = {
    root: "src/",
    buildOptions: {
        out: "static/js/"
    },
    plugins: [
        '@snowpack/plugin-typescript'
    ]
};
