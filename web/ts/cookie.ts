function getCookie(name: string) {
  let state = 0;
  let length = document.cookie.length;
  let found = false;
  let key = "";
  let value = "";
  function checkKey() {
    if (key == name) {
      found = true;
    } else {
      key = "";
      value = "";
      state = 2;
    }
  }
  for (let i = 0; i < length && !found; i++) {
    let c = document.cookie.charAt(i);
    switch (state) {
      case 0: //parsing from the start of the cookie
        if (c == '=') {
          state = 1;
        } else if (c == ';') {
          value = key;
          key = "";
          checkKey();
        } else {
          key = key + c;
        }
        break;
      case 1: //parsing from after the '=' of a cookie
        if (c == ';') {
          checkKey();
        } else {
          value = value + c;
        }
        break;
      case 2: //chew through whitespace after semicolon
        if (c == '=') {
          key = "";
          state = 1;
        } else if (c == ';') {
          key = "";
          value = "";
          checkKey();
        } else if (c != ' ' && c != '\t') {
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

function deleteCookie(name: string) {
  document.cookie = name + '=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
}