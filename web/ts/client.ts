/*======================================== *
 * This file defines a Doomsday API client *
 * ========================================*/

/**
 * Enumeration of supported auth methods
 */
enum AuthMethod { NONE, USERPASS };

/**
 * Contains error information returned from an HTTP API call
 */
class APIError {
	readonly error: string;
	readonly code: number;

	constructor(readonly errorMessage: string, readonly returnCode: number) {
		this.error = errorMessage;
		this.code = returnCode;
	}
}

/**
 * An HTTP Client to the Doomsday API
 */
class Doomsday {
	private doRequest(method: string, path: string, data?: object): JQuery.Promise<any> {
		return $.ajax({
			method: method,
			url: path,
			dataType: "json",
			data: (data ? JSON.stringify(data) : undefined)
		}).promise();
	}

	private handleFailure(jqXHR: JQuery.jqXHR<any>, textStatus: string): never {
		throw new APIError(textStatus, jqXHR.status);
	}

	fetchAuthType(): Promise<any> {
		return this.doRequest("GET", "/v1/info")
			.then(
				data => (data.auth_type == "Userpass" ? AuthMethod.USERPASS : AuthMethod.NONE),
				this.handleFailure
			);
	}

	authUser(username: string, password: string): Promise<any> {
		return this.doRequest("POST", "/v1/auth", {
			username: username,
			password: password
		})
			.then(
				() => { },
				this.handleFailure
			);
	}

	fetchCerts(): Promise<any> {
		return this.doRequest("GET", "/v1/cache")
			.then(data => data.content,
				this.handleFailure);
	}
}
