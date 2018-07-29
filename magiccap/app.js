// This code is a part of MagicCap which is a MPL-2.0 licensed project.
// Copyright (C) Jake Gealer <jake@gealer.email> 2018.
// Copyright (C) Rhys O'Kane <SunburntRock89@gmail.com> 2018.

const configtemplate = {
	hotkey: "",
	novus_token: "",
};

const { stat, writeJSON } = require("fs-nextra");

(async() => {
	let fileExists = await stat(`${require("os").homedir()}/magiccap.json`).catch(async() => {
		writeJSON(`${require("os").homedir()}/magiccap.json`, configtemplate).catch(async() => {
			throw new Error("Could not find or create config file.");
		});
	});
})();

(async() => {
	let fileExists = await stat(`${require("os").homedir()}/magiccap_captures.json`).catch(async() => {
		writeJSON(`${require("os").homedir()}/magiccap_captures.json`, { captures: [] }).catch(async() => {
			throw new Error("Could not find or create capture logging file.");
		});
	});
})();

const captures = global.captures = require(`${require("os").homedir()}/magiccap_captures.json`);

const config = global.config = require(`${require("os").homedir()}/magiccap.json`);
const capture = require("./capture.js");
const { app, Tray, Menu, dialog, Notification } = require("electron");

async function runCapture() {
	let filename = capture.createCaptureFilename();
	let result;
	try {
		result = await capture.handleScreenshotting(filename);
		Notification("MagicCap", { body: result });
	} catch (err) {
		await capture.logUpload(filename, false, null, null);
		dialog.showErrorBox("MagicCap", `${err}`);
	}
}
// Runs the capture.

function initialiseScript() {
	const tray = new Tray(`${__dirname}/icons/taskbar.png`);
	const contextMenu = Menu.buildFromTemplate([
		{ label: "Exit", type: "normal", role: "quit" },
	]);
	tray.setContextMenu(contextMenu);
}
// Initialises the script.

app.on("ready", initialiseScript);
// The app is ready to rock!
