/// <reference path="../color.ts"/>
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
            let label = this.durationString(maxDays - 1);
            lists.push({
              header: label,
              cutoff: now + (maxDays * 86400),
              color: this.cardColor(maxDays - 1),
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


  private durationString(days: number) {
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
        ret = ret + ", " + this.durationString(remaining_days);
      }
      return ret;
    }
  }

  private cardColor(days: number) {
    if (days < 0) {
      return Colors.Black;
    } else if (days < 3) {
      return Colors.Red;
    } else if (days < 7) {
      return Colors.Red.shift(Colors.Orange, 1 - ((7 - days) / 4));
    } else if (days < 14) {
      return Colors.Orange.shift(Colors.OrangeYellow, 1 - ((14 - days) / 7));
    } else if (days < 21) {
      return Colors.OrangeYellow.shift(Colors.Yellow, 1 - ((21 - days) / 7));
    } else if (days < 28) {
      return Colors.Yellow.shift(Colors.Green, 1 - ((28 - days) / 7));
    }

    return Colors.Green;
  }
}
