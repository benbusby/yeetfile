import {hashBlake2b} from "./crypto.js";
import {fetchLocalStorageString, setLocalStorageString} from "./localstorage.js";

const init = () => {
    let banner = document.getElementById("yeetfile-banner");
    let bannerString = document.getElementById("banner-string");
    let closeBanner = document.getElementById("close-banner");

    let bannerHash = new TextDecoder().decode(hashBlake2b(32, bannerString.innerText));
    if (!fetchLocalStorageString(bannerHash, "")) {
        banner.style.display = "block";
    }

    closeBanner.addEventListener("click", () => {
        banner.remove();
        setLocalStorageString(bannerHash, "1");
    });
}

if (document.readyState !== "loading") {
    init();
} else {
    document.addEventListener("DOMContentLoaded", () => {
        init();
    });
}