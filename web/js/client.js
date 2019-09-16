/*======================================== *
 * This file defines a Doomsday API client *
 * ========================================*/

/**
 * Enumeration of supported auth methods
 */
const AuthMethod = {
	NONE: 0,
	USERPASS: 1
}

/**
 * Contains error information returned from an HTTP API call
 */
function APIError(error, code) {
	this._error = error
	this._code = code;

	this.getError = () => this._error;
	this.getCode = () => this._code;
}

/**
 * An HTTP Client to the Doomsday API
 */
function Doomsday() {
	this._doRequest = (method, path, data) => {
		let dataStr = undefined;
		if (data) { dataStr = JSON.stringify(data); }
		return $.ajax({
			method: method,
			url: path,
			dataType: "json",
			data: dataStr
		})
	};

	this._handleFailure = (jqXHR, textStatus) => {
		throw new APIError(textStatus, jqXHR.status);
	}

	this.fetchAuthType = () => {
		return this._doRequest("GET", "/v1/info")
		.then(
			data => (data.auth_type == "Userpass" ? AuthMethod.USERPASS : AuthMethod.NONE),
			this._handleFailure
		);
	};

	this.authUser = (username, password) => {
		return this._doRequest("POST", "/v1/auth", {
			username: username, 
			password: password
		})
		.then(
			() => {},
			this._handleFailure
		);
	};

	this.fetchCerts = () => {
		return this._doRequest("GET", "/v1/cache")
		.then(
			data => data.content,
			this._handleFailure
		);
	};
}


