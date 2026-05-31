// Navigate to the Concio login page if not already inside the app.
if (document.querySelector('.chat-list-inner')) return false;
if (location.href.indexOf('/starise/im') < 0) {
    window.location.href = 'https://octonhq01.starise.network/starise/im/#/Login';
    return true;
}
return false;
