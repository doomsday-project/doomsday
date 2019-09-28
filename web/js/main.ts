/// <reference path="./client.ts"/>

//Because Lens.js adds template to $()
interface JQuery {
	template(template: string, options?: object): any;
}

function durationString(days: number) {
	if (days < 0) {
		return "THE DAWN OF TIME";
	} else if (days == 0) {
		return "NOW";
	} else if (days == 1) {
		return "1 DAY";
	} else if (days < 7) {
		return days + " DAYS";
	} else {
		var weeks = Math.floor(days / 7);
		var remaining_days = days - (weeks * 7);
		var ret = weeks + " WEEKS";
		if (weeks == 1) {
			ret = "1 WEEK";
		}
		if (remaining_days > 0) {
			ret = ret + ", " + durationString(remaining_days);
		}
		return ret;
	}
}

function cardColor(days: number) {
	if (days < 0) {
		return [0, 0, 0];
	} else if (days < 3) {
		return [229, 53, 69]; //red
	} else if (days < 7) {
		return colorShift([229, 53, 69], [253, 126, 20], (7 - days) / 4);
	} else if (days < 14) {
		return colorShift([253, 126, 20], [255, 193, 7], (14 - days) / 7);
	} else if (days < 21) {
		return colorShift([255, 193, 7], [200, 185, 15], (21 - days) / 7);
	} else if (days < 28) {
		return colorShift([200, 185, 15], [40, 167, 69], (28 - days) / 7);
	}

	return [40, 167, 69];
}

type Color = [number, number, number]

function colorShift(end: Color, start: Color, percent: number) {
	return [
		start[0] + ((end[0] - start[0]) * percent),
		start[1] + ((end[1] - start[1]) * percent),
		start[2] + ((end[2] - start[2]) * percent)
	];
}

function updateCertList() {
	doomsday.fetchCerts()
		.then(content => {
			var now = new Date().getTime() / 1000;

			var lists = [];

			for (var i = 0; i < content.length; i++) {
				var cert = content[i];
				if (cert.not_after - now > 7776000) {
					break;
				}

				if (lists.length == 0 || cert.not_after > lists[lists.length - 1].cutoff) {
					var maxDays = Math.max(0, Math.ceil((cert.not_after - now) / 86400));
					var label = durationString(maxDays - 1);
					lists.push({
						header: label,
						cutoff: now + (maxDays * 86400),
						color: cardColor(maxDays - 1),
						certs: [cert]
					});
				} else {
					lists[lists.length - 1].certs.push(cert);
				}
			}

			console.log(lists.length);

			if (lists.length == 0) {
				$("#certs").template("no-certs-page");
				return;
			}

			$("#certs").template("cert-list-group", { lists: lists });
			$("#certs").show();
			certUpdateID = setTimeout(updateCertList, 60 * 1000);
		})
		.catch(e => {
			if (e.error == "error" && e.code == 401) {
				deleteCookie('doomsday-token');
				gotoLogin("Your session has expired");
			} else {
				gotoLogin("Something went wrong!");
			}
		});
}

let doomsday = new Doomsday();
let NORMAL_HAMBURGER_WIDTH;
let NORMAL_HAMBURGER_HEIGHT;
let HAMBURGER_BOX_PADDING;

$(document).ready(function () {
	let hamburgerBox = $('#hamburger-box');
	NORMAL_HAMBURGER_WIDTH = hamburgerBox.width();
	NORMAL_HAMBURGER_HEIGHT = $('#hamburger').height();
	HAMBURGER_BOX_PADDING = hamburgerBox.innerWidth() - NORMAL_HAMBURGER_WIDTH;

	doomsday.fetchAuthType()
		.then(authType => {
			if (authType == AuthMethod.NONE) {
				let logout_button = $('#logout-button');
				logout_button.addClass('hamburger-menu-button-inactive');
				logout_button.removeClass('navbar-button hamburger-menu-button');
				logout_button.mouseover(function () { logout_button.text('auth is turned off'); });
				logout_button.mouseout(function () { logout_button.text('logout'); });
			} else {
				$('#logout-button').click(function () {
					closeHamburgerMenu();
					handleLogout();
				});
			}
			if (authType == AuthMethod.USERPASS && getCookie('doomsday-token') == "") {
				gotoLogin();
			} else {
				gotoDashboard();
			}
		})
		.catch(() => { console.log("Something went wrong!"); });
});

let certUpdateID = -1;

function handleLogin(e) {
	let username = ($('input[name=username]').val() as string);
	let password = ($('input[name=password]').val() as string);
	doomsday.authUser(username, password)
		.then(() => { gotoDashboard(); })
		.catch(e => {
			if (e.error == "error" && e.code == 401) {
				gotoLogin("The username and password did not match");
			}
			else { gotoLogin("Something went wrong!"); }
		});
	return false;
}

function handleLogout() {
	deleteCookie('doomsday-token');
	gotoLogin();
}

function gotoLogin(message?: string) {
	clearTimeout(certUpdateID);
	certUpdateID = -1;
	$("#certs").hide();
	$("#hamburger-box").hide();

	var templateParams: { error_message?: string } = {};
	if (typeof message !== 'undefined') {
		templateParams.error_message = message;
	}
	$("#login").template("login-page", templateParams);

	$("#login-form").submit(handleLogin);
	$("#login-form input[name=password]").val("");
	$("#login").show();
}

function gotoDashboard() {
	$("#login").hide();
	$("#login-form").off("submit");
	$("#certs").show();
	$('#hamburger-box').show();

	updateCertList();
}

const FRAMERATE = 42;
const FRAME_INTERVAL = 1000 / FRAMERATE;

const NO_ANIM = -1;

let hamburgerMenuOpen = false;

let currentHamburgerMenuOpenness = 0;

function setHamburgerMenuOpenness(percentage: number) {
	let menu = $('#hamburger-menu');
	//The +1 is for the 1px wide border
	let menuWidth = menu.innerWidth() + 1;
	let desiredShift = menuWidth * percentage;

	menu.css('left', (-menuWidth + desiredShift) + "px");

	let boxWidth = Math.max(desiredShift - (1 + HAMBURGER_BOX_PADDING), NORMAL_HAMBURGER_WIDTH);
	let boxHeight = NORMAL_HAMBURGER_HEIGHT - (percentage * (NORMAL_HAMBURGER_HEIGHT * 0.1));
	$('#hamburger-box').css('width', boxWidth + "px");
	$('#hamburger').css('height', boxHeight + "px");
	currentHamburgerMenuOpenness = percentage;
}

let menuOpenAnimID = NO_ANIM;

function hamburgerMenuSlide(start: number, end: number) {
	if (menuOpenAnimID != NO_ANIM) {
		clearInterval(menuOpenAnimID);
	}
	let duration = 0.2; //in seconds
	let totalDelta = end - start;
	let lastAnimTime = new Date().getTime();
	return function () {
		let now = new Date().getTime();
		let timeDelta = now - lastAnimTime;
		let updatePercentage = (duration * 1000) / timeDelta;
		let frameDelta = totalDelta / updatePercentage;
		lastAnimTime = now;

		let desiredOpenness = currentHamburgerMenuOpenness + frameDelta;
		if ((totalDelta >= 0 && desiredOpenness >= end) || (totalDelta < 0 && desiredOpenness <= end)) {
			desiredOpenness = end;
			clearInterval(menuOpenAnimID);
			menuOpenAnimID = NO_ANIM;
		}

		setHamburgerMenuOpenness(desiredOpenness);
	};
}

function openHamburgerMenu() {
	menuOpenAnimID = setInterval(hamburgerMenuSlide(0, 1), FRAME_INTERVAL);
	hamburgerMenuOpen = true;
}

function closeHamburgerMenu() {
	menuOpenAnimID = setInterval(hamburgerMenuSlide(1, 0), FRAME_INTERVAL);
	hamburgerMenuOpen = false;
}

function toggleHamburgerMenu() {
	if (hamburgerMenuOpen) {
		closeHamburgerMenu();
	} else {
		openHamburgerMenu();
	}
}

$('#hamburger-box').click(function () {
	toggleHamburgerMenu();
});


function getCookie(name: string) {
	let state = 0;
	let length = document.cookie.length;
	let found = false;
	let key = "";
	let value = "";
	function checkKey() {
		if (key == name) {
			found = true;
		} else {
			key = "";
			value = "";
			state = 2;
		}
	}
	for (let i = 0; i < length && !found; i++) {
		let c = document.cookie.charAt(i);
		switch (state) {
			case 0: //parsing from the start of the cookie
				if (c == '=') {
					state = 1;
				} else if (c == ';') {
					value = key;
					key = "";
					checkKey();
				} else {
					key = key + c;
				}
				break;
			case 1: //parsing from after the '=' of a cookie
				if (c == ';') {
					checkKey();
				} else {
					value = value + c;
				}
				break;
			case 2: //chew through whitespace after semicolon
				if (c == '=') {
					key = "";
					state = 1;
				} else if (c == ';') {
					key = "";
					value = "";
					checkKey();
				} else if (c != ' ' && c != '\t') {
					key = c;
					state = 0;
				}
				break;
		}
	}

	if (!found && key != name) {
		value = "";
	}

	return value;
}

function deleteCookie(name: string) {
	document.cookie = name + '=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
}
