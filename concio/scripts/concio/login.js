// Submits the Concio login form.
//   arguments[0] = password  (CONCIO_PASSWORD)
//   arguments[1] = user ID   (CONCIO_USER)
// Returns true if a login was performed, false if already inside the app,
// null if the form wasn't found or the user ID wasn't supplied.
const pw   = arguments[0];
const user = arguments[1];
if (document.querySelector('.chat-list-inner')) return false;
if (!user) return null;          // CONCIO_USER was not supplied
const userInp = document.querySelector('#loginUserInp');
const passInp = document.querySelector('#loginPassInp');
const btn = document.querySelector('#loginFormLoginBtn');
if (!userInp || !passInp || !btn) return null;

function set(el, v) {
    const proto = Object.getPrototypeOf(el);
    const setter = Object.getOwnPropertyDescriptor(proto, 'value').set;
    setter.call(el, v);
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
}

set(userInp, user);
set(passInp, pw);
const opts = { bubbles: true, cancelable: true, view: window, button: 0 };
btn.dispatchEvent(new MouseEvent('mousedown', opts));
btn.dispatchEvent(new MouseEvent('mouseup', opts));
btn.dispatchEvent(new MouseEvent('click', opts));
return true;
