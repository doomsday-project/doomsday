/// <reference path="../color.ts"/>
class DashboardPage extends PageBase {
  private certUpdateID: number;
  private certsElement: JQuery;
  private showMoreButton: JQuery;
  private certs: CertificateList;
  private shouldShowAll: boolean;

  static readonly DEFAULT_EXPIRY_CUTOFF: number = 7776000;
  constructor() {
    super();
    this.certUpdateID = -1;
    this.certsElement = $("#certs");
  }

  initialize(): void {
    this.certsElement.show();
    this.certs = new CertificateList();

    this.updateCertList();
  }

  teardown(): void {
    clearTimeout(this.certUpdateID);
    this.certUpdateID = -1;
    this.certsElement.hide();
    this.showMoreButton.off();
  }

  private updateCertList() {
    this.ctx.client.fetchCerts()
      .then((content: Array<Certificate>) => {
        this.certs = new CertificateList(content);
        this.repaint();
        this.certUpdateID = setTimeout(this.updateCertList.bind(this), 60 * 1000);
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

  private repaint() {
    let now = new Date().getTime() / 1000;
    let lists = [];
    let certsToDisplay = this.certs;
    if (!this.shouldShowAll) {
      certsToDisplay = this.certs.expiresWithin(DashboardPage.DEFAULT_EXPIRY_CUTOFF)
    }
    for (let cert of certsToDisplay) {
      if (lists.length == 0 || cert.notAfter > lists[lists.length - 1].cutoff) {
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

    if (lists.length == 0) {
      this.certsElement.template("no-certs-page");
      return;
    }

    this.certsElement.template("cert-list-group", { lists: lists });
    this.showMoreButton = $("#certs-show-more");
    this.showMoreButton.on("click", (e: JQuery.Event) => {
      e.preventDefault();
      this.showMoreButton.prop("disabled", true);
      this.showMoreButton.html("working...");
      this.shouldShowAll = !this.shouldShowAll;
      this.repaint();
      this.showMoreButton = $("#certs-show-more");
      this.showMoreButton.html("show " + (this.shouldShowAll ? "less" : "all"));
      this.showMoreButton.prop("disabled", false);
      return false;
    })
    this.certsElement.show();
  }


  private durationString(days: number): string {
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

  private cardColor(days: number): Color {
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

/**
 * A list of Certificates
 *
 * @remarks
 * Given input must currently be presented in a sorted order
 */
class CertificateList {
  private storage: Array<Certificate>
  private expMax: number; //inclusive

  constructor(initial?: Array<Certificate>) {
    this.storage = [];
    if (initial) {
      this.storage = initial;
    }
  }

  [Symbol.iterator]() {
    const now = new Date().getTime() / 1000;
    const maxUntil: number =
      (typeof this.expMax === 'number' ? this.expMax : Number.MAX_VALUE);
    let idx: number = 0;
    return {
      next: () => {
        let isDone = () => {
          if (idx >= this.storage.length) {
            return true;
          }
          if (this.storage[idx].notAfter - now > maxUntil) {
            return true;
          }
          return false;
        }
        let done: boolean = isDone();
        return {
          done: done,
          value: done ? undefined : this.storage[idx++]
        }
      }
    }
  }

  get length(): number { return this.storage.length; }

  /**
   * Returns a new CertificateStore which, when iterated over, will only yield
   * the elements of the CertificateStore which expire within the given timespan.
   *
   * @param duration Number of seconds from now a cert must expire within to
   *   be returned.
   */
  expiresWithin(duration: number): CertificateList {
    let ret: CertificateList = new CertificateList(this.storage);
    ret.expMax = duration;
    return ret;
  }
}
