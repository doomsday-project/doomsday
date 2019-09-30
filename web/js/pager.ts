/// <reference path="./client.ts"/>
class Pager {
  private ctx: PageContext;
  private curPage: Page;

  constructor(client: Doomsday) {
    this.ctx = {
      client: client,
      hamburgerMenu: $("#hamburger-box"),
      pager: this
    };
  }

  display(page: Page): void {
    if (this.curPage != null) {
      this.curPage.teardown();
    }

    this.curPage = page;
    page.setContext(this.ctx);
    if (page.settings && page.settings.hideHamburgerMenu) {
      this.ctx.hamburgerMenu.hide()
    } else {
      this.ctx.hamburgerMenu.show();
    }

    page.initialize();
  }
}

class PageContext {
  client: Doomsday;
  hamburgerMenu: JQuery;
  pager: Pager;
}


interface Page {
  settings: PageSettings;
  initialize(...args: any): void;
  teardown(): void;
  setContext(ctx: PageContext): void;
}

abstract class PageBase implements Page {
  protected _settings: PageSettings;
  protected ctx: PageContext;

  abstract initialize(): void;
  abstract teardown(): void;
  setContext(ctx: PageContext) {
    this.ctx = ctx;
  }
  get settings(): PageSettings {
    return this._settings;
  }
}

class PageSettings {
  hideHamburgerMenu: boolean;
}