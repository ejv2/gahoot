// Snowpack Configuration File
// See all supported options: https://www.snowpack.dev/reference/configuration

/** @type {import("snowpack").SnowpackUserConfig } */
module.exports = {
    root: "./.genjs/",
    exclude: ["**/node_modules/**/*", "*.ts", "*.tsbuildinfo"],
    buildOptions: {
        out: "./static/js/"
    },
};
