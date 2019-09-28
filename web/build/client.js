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
