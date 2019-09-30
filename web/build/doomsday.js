var AuthMethod;
(function (AuthMethod) {
    AuthMethod[AuthMethod["NONE"] = 0] = "NONE";
    AuthMethod[AuthMethod["USERPASS"] = 1] = "USERPASS";
})(AuthMethod || (AuthMethod = {}));
;
class APIError {
    constructor(errorMessage, returnCode) {
        this.errorMessage = errorMessage;
        this.returnCode = returnCode;
        this.error = errorMessage;
        this.code = returnCode;
    }
}
class Doomsday {
    doRequest(method, path, data) {
        return $.ajax({
            method: method,
            url: path,
            dataType: "json",
            data: (data ? JSON.stringify(data) : undefined)
        }).promise();
    }
    handleFailure(jqXHR, textStatus) {
        throw new APIError(textStatus, jqXHR.status);
    }
    fetchAuthType() {
        return this.doRequest("GET", "/v1/info")
            .then(data => (data.auth_type == "Userpass" ? AuthMethod.USERPASS : AuthMethod.NONE), this.handleFailure);
    }
    authUser(username, password) {
        return this.doRequest("POST", "/v1/auth", {
            username: username,
            password: password
        })
            .then(() => { }, this.handleFailure);
    }
    fetchCerts() {
        return this.doRequest("GET", "/v1/cache")
            .then(data => data.content, this.handleFailure);
    }
}
function getCookie(name) {
    let state = 0;
    let length = document.cookie.length;
    let found = false;
    let key = "";
    let value = "";
    function checkKey() {
        if (key == name) {
            found = true;
        }
        else {
            key = "";
            value = "";
            state = 2;
        }
    }
    for (let i = 0; i < length && !found; i++) {
        let c = document.cookie.charAt(i);
        switch (state) {
            case 0:
                if (c == '=') {
                    state = 1;
                }
                else if (c == ';') {
                    value = key;
                    key = "";
                    checkKey();
                }
                else {
                    key = key + c;
                }
                break;
            case 1:
                if (c == ';') {
                    checkKey();
                }
                else {
                    value = value + c;
                }
                break;
            case 2:
                if (c == '=') {
                    key = "";
                    state = 1;
                }
                else if (c == ';') {
                    key = "";
                    value = "";
                    checkKey();
                }
                else if (c != ' ' && c != '\t') {
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
function deleteCookie(name) {
    document.cookie = name + '=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
}
class Pager {
    constructor(client) {
        this.ctx = {
            client: client,
            hamburgerMenu: $("#hamburger-box"),
            pager: this
        };
    }
    display(page) {
        if (this.curPage != null) {
            this.curPage.teardown();
        }
        this.curPage = page;
        page.setContext(this.ctx);
        if (page.settings && page.settings.hideHamburgerMenu) {
            this.ctx.hamburgerMenu.hide();
        }
        else {
            this.ctx.hamburgerMenu.show();
        }
        page.initialize();
    }
}
class PageContext {
}
class PageBase {
    setContext(ctx) {
        this.ctx = ctx;
    }
    get settings() {
        return this._settings;
    }
}
class PageSettings {
}
class LoginPage extends PageBase {
    constructor(message) {
        super();
        this.message = message;
        this._settings = {
            hideHamburgerMenu: true
        };
    }
    initialize() {
        var templateParams = {};
        if (this.message) {
            templateParams.error_message = this.message;
        }
        this.login = $("#login");
        this.login.template("login-page", templateParams);
        this.loginForm = $("#login-form");
        this.loginFormUsername = $("#login-form input[name=username]");
        this.loginFormPassword = $("#login-form input[name=password]");
        this.loginForm.submit(this.getLoginHandler());
        this.login.show();
    }
    teardown() {
        this.login.hide();
        this.loginForm.off("submit");
    }
    getLoginHandler() {
        let self = this;
        return function (e) {
            let username = self.loginFormUsername.val();
            let password = self.loginFormPassword.val();
            self.loginFormPassword.val("");
            self.ctx.client.authUser(username, password)
                .then(() => {
                self.ctx.pager.display(new DashboardPage());
            })
                .catch(e => {
                if (e.error == "error" && e.code == 401) {
                    self.ctx.pager.display(new LoginPage("The username and password did not match"));
                }
                else {
                    self.ctx.pager.display(new LoginPage("Something went wrong!"));
                }
            });
            return false;
        };
    }
}
class DashboardPage extends PageBase {
    constructor() {
        super();
        this.certUpdateID = -1;
        this.certsElement = $("#certs");
    }
    initialize() {
        this.certsElement.show();
        this.updateCertList();
    }
    teardown() {
        clearTimeout(this.certUpdateID);
        this.certUpdateID = -1;
        this.certsElement.hide();
    }
    updateCertList() {
        this.ctx.client.fetchCerts()
            .then(content => {
            let now = new Date().getTime() / 1000;
            let lists = [];
            for (var i = 0; i < content.length; i++) {
                let cert = content[i];
                if (cert.not_after - now > 7776000) {
                    break;
                }
                if (lists.length == 0 || cert.not_after > lists[lists.length - 1].cutoff) {
                    let maxDays = Math.max(0, Math.ceil((cert.not_after - now) / 86400));
                    let label = durationString(maxDays - 1);
                    lists.push({
                        header: label,
                        cutoff: now + (maxDays * 86400),
                        color: cardColor(maxDays - 1),
                        certs: [cert]
                    });
                }
                else {
                    lists[lists.length - 1].certs.push(cert);
                }
            }
            if (lists.length == 0) {
                this.certsElement.template("no-certs-page");
                return;
            }
            this.certsElement.template("cert-list-group", { lists: lists });
            this.certsElement.show();
            this.certUpdateID = setTimeout(this.updateCertList, 60 * 1000);
        })
            .catch(e => {
            if (e.error == "error" && e.code == 401) {
                deleteCookie('doomsday-token');
                this.ctx.pager.display(new LoginPage("Your session has expired"));
            }
            else {
                this.ctx.pager.display(new LoginPage("Something went wrong!"));
            }
        });
    }
}
function durationString(days) {
    if (days < 0) {
        return "THE DAWN OF TIME";
    }
    else if (days == 0) {
        return "NOW";
    }
    else if (days == 1) {
        return "1 DAY";
    }
    else if (days < 7) {
        return days + " DAYS";
    }
    else {
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
function cardColor(days) {
    if (days < 0) {
        return [0, 0, 0];
    }
    else if (days < 3) {
        return [229, 53, 69];
    }
    else if (days < 7) {
        return colorShift([229, 53, 69], [253, 126, 20], (7 - days) / 4);
    }
    else if (days < 14) {
        return colorShift([253, 126, 20], [255, 193, 7], (14 - days) / 7);
    }
    else if (days < 21) {
        return colorShift([255, 193, 7], [200, 185, 15], (21 - days) / 7);
    }
    else if (days < 28) {
        return colorShift([200, 185, 15], [40, 167, 69], (28 - days) / 7);
    }
    return [40, 167, 69];
}
function colorShift(end, start, percent) {
    return [
        start[0] + ((end[0] - start[0]) * percent),
        start[1] + ((end[1] - start[1]) * percent),
        start[2] + ((end[2] - start[2]) * percent)
    ];
}
let NORMAL_HAMBURGER_WIDTH;
let NORMAL_HAMBURGER_HEIGHT;
let HAMBURGER_BOX_PADDING;
$(document).ready(function () {
    let hamburgerBox = $('#hamburger-box');
    NORMAL_HAMBURGER_WIDTH = hamburgerBox.width();
    NORMAL_HAMBURGER_HEIGHT = $('#hamburger').height();
    HAMBURGER_BOX_PADDING = hamburgerBox.innerWidth() - NORMAL_HAMBURGER_WIDTH;
    let doomsday = new Doomsday();
    let pager = new Pager(doomsday);
    doomsday.fetchAuthType()
        .then(authType => {
        if (authType == AuthMethod.NONE) {
            let logout_button = $('#logout-button');
            logout_button.addClass('hamburger-menu-button-inactive');
            logout_button.removeClass('navbar-button hamburger-menu-button');
            logout_button.mouseover(function () { logout_button.text('auth is turned off'); });
            logout_button.mouseout(function () { logout_button.text('logout'); });
        }
        else {
            $('#logout-button').click(function () {
                closeHamburgerMenu();
                deleteCookie('doomsday-token');
                pager.display(new LoginPage());
            });
        }
        if (authType == AuthMethod.USERPASS && getCookie('doomsday-token') == "") {
            pager.display(new LoginPage());
        }
        else {
            pager.display(new DashboardPage());
        }
    })
        .catch(() => { console.log("Something went wrong!"); });
});
const FRAMERATE = 42;
const FRAME_INTERVAL = 1000 / FRAMERATE;
const NO_ANIM = -1;
let hamburgerMenuOpen = false;
let currentHamburgerMenuOpenness = 0;
function setHamburgerMenuOpenness(percentage) {
    let menu = $('#hamburger-menu');
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
function hamburgerMenuSlide(start, end) {
    if (menuOpenAnimID != NO_ANIM) {
        clearInterval(menuOpenAnimID);
    }
    let duration = 0.2;
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
    }
    else {
        openHamburgerMenu();
    }
}
$('#hamburger-box').click(function () {
    toggleHamburgerMenu();
});
