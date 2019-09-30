class DashboardPage extends PageBase {
  private certUpdateID: number;
  private certsElement: JQuery;
  constructor() {
    super();
    this.certUpdateID = -1;
    this.certsElement = $("#certs");
  }

  initialize(): void {
    this.certsElement.show();
    this.updateCertList();
  }

  teardown(): void {
    clearTimeout(this.certUpdateID);
    this.certUpdateID = -1;
    this.certsElement.hide();
  }
  private updateCertList() {
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
          } else {
            lists[lists.length - 1].certs.push(cert);
          }
        }

        //console.log(lists.length);

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
        } else {
          this.ctx.pager.display(new LoginPage("Something went wrong!"));
        }
      });
  }
}