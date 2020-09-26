class Color {
  readonly red: number;
  readonly green: number;
  readonly blue: number;
  constructor(red: number, green: number, blue: number) {
    this.red = red;
    this.green = green;
    this.blue = blue;
  }
  get 0(): number { return this.red; }
  get 1(): number { return this.green; }
  get 2(): number { return this.blue; }
  hex(): string {
    return "#" + this.cAsHex(this.red) + this.cAsHex(this.green) + this.cAsHex(this.blue);
  }

  shift(c2: Color, percent: number): Color {
    return new Color(
      this.red + ((c2.red - this.red) * percent),
      this.green + ((c2.green - this.green) * percent),
      this.blue + ((c2.blue - this.blue) * percent)
    );
  }

  private cAsHex(c: number): string {
    var hex = c.toString(16);
    return hex.length == 1 ? "0" + hex : hex;;
  }
}

namespace Colors {
  export const Black = new Color(0, 0, 0);
  export const Red = new Color(229, 53, 69);
  export const Orange = new Color(253, 126, 20);
  export const OrangeYellow = new Color(255, 193, 7);
  export const Yellow = new Color(200, 185, 15);
  export const Green = new Color(40, 167, 69);
}