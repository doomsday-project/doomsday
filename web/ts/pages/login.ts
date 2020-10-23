class LoginPage extends PageBase {
  private readonly message;
  private login: JQuery;
  private loginForm: JQuery;
  private loginFormUsername: JQuery;
  private loginFormPassword: JQuery;
  constructor(message?: string) {
    super();
    this.message = message;
    this._settings = {
      hideHamburgerMenu: true
    };
  }

  initialize(): void {
    var templateParams: { error_message?: string } = {};
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

  teardown(): void {
    this.login.hide();
    this.loginForm.off("submit");
  }

  private getLoginHandler() {
    let self = this;
    return function (e: JQuery.Event) {
      let username = (self.loginFormUsername.val() as string);
      let password = (self.loginFormPassword.val() as string);
      self.loginFormPassword.val("");
      self.ctx.client.authUser(username, password)
        .then(() => {
          self.ctx.pager.display(new DashboardPage());
        })
        .catch(e => {
          if (e.code == 401) {
            self.ctx.pager.display(new LoginPage("The username and password did not match"));
          }
          else {
            self.ctx.pager.display(new LoginPage("Something went wrong!"));
            console.log(`Something went wrong: ${e.errorMessage}`);
          }
        });
      return false;
    }
  }
}