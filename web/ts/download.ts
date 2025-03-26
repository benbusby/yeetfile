import * as crypto from "./crypto.js";
import * as transfer from "./transfer.js";
import * as interfaces from "./interfaces.js";
import {Endpoints} from "./endpoints.js";
import {TextIDPrefix} from "./constants.js";

let timeoutInterval;

const init = () => {
    let xhr = new XMLHttpRequest();
    let id = window.location.pathname.split("/").slice(-1)[0];
    let url = Endpoints.format(Endpoints.DownloadSendFileMetadata, id);
    xhr.open("GET", url, true);
    xhr.setRequestHeader("Content-Type", "application/json");

    xhr.onreadystatechange = async () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            let download = new interfaces.DownloadResponse(xhr.responseText);
            await handleMetadata(download);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            alert(`Error ${xhr.status}: ${xhr.responseText}`);
            window.location.assign("/");
        }
    };

    xhr.send();
};

const handleMetadata = async (download: interfaces.DownloadResponse) => {
    // Attempt to decrypt without a password first
    let secret = getSecret();

    let key = await crypto.importKey(fromURLSafeBase64(secret));
    decryptName(key, download.name).then(result => {
        showDownload(result, download, key);
    }).catch(err => {
        console.warn(err);
        promptPassword(download);
    });
}

const getSecret = (): string => {
    let params = new URLSearchParams(document.location.search);
    let secret = params.get("secret");
    if (!secret) {
        return location.hash.slice(1);
    }

    return secret;
}

const showDownload = (name: string, download: interfaces.DownloadResponse, key: CryptoKey) => {
    let downloadBtn = document.getElementById("download-nopass") as HTMLButtonElement;

    let loading = document.getElementById("loading");
    loading.style.display = "none";

    let passwordPrompt = document.getElementById("password-prompt-div");
    passwordPrompt.style.display = "none";

    let nameSpan = document.getElementById("name");
    if (download.id.startsWith(`${TextIDPrefix}_`)) {
        nameSpan.textContent = "N/A (text-only)";
        downloadBtn.innerText = "Show Text Content";
    } else {
        nameSpan.textContent = name;
    }

    let expiration = document.getElementById("expiration");
    console.log(download.expiration.getFullYear())
    if (download.expiration.getFullYear() > 2100) {
        expiration.textContent = "Never";
    } else {
        expiration.textContent = calcTimeRemaining(download.expiration);
        timeoutInterval = window.setInterval(() => {
            expiration.textContent = calcTimeRemaining(download.expiration);
        }, 1000);
    }

    let downloads = document.getElementById("downloads");
    if (download.downloads === -1) {
        downloads.textContent = "Unlimited";
    } else {
        downloads.textContent = String(download.downloads);
    }

    let size = document.getElementById("size");
    size.textContent = calcFileSize(download.size);

    let downloadDiv = document.getElementById("download-prompt-div");
    downloadDiv.style.display = "inherit";

    downloadBtn.addEventListener("click", () => {
        downloadBtn.disabled = true;
        downloadBtn.innerText = "Downloading...";

        const downloadCallback = success => {
            if (success) {
                downloadBtn.style.display = "none";
                if (download.downloads > 0) {
                    let newDownloadCount = download.downloads - 1;
                    downloads.textContent = String(newDownloadCount);

                    if (newDownloadCount === 0) {
                        downloads.textContent += " (deleted)";
                        expiration.textContent = "---";
                        clearInterval(timeoutInterval);
                    }
                }
            } else {
                downloadBtn.disabled = false;
            }
        }

        if (download.id.startsWith("text_")) {
            downloadText(download, key, downloadCallback);
        } else {
            transfer.downloadSentFile(name, download, key, downloadCallback, () => {});
        }
    })
}

const decryptName = async (key, name) => {
    let nameBytes = hexToBytes(name);
    return await crypto.decryptString(key, nameBytes);
}

const updatePasswordBtn = (txt, disabled) => {
    let btn = document.getElementById("submit") as HTMLButtonElement;
    btn.value = txt;
    btn.textContent = txt;
    btn.disabled = disabled;
}

const setFormEnabled = on => {
    let fieldset = document.getElementById("download-fieldset") as HTMLFieldSetElement;
    fieldset.disabled = !on;
}

const calcTimeRemaining = expiration => {
    let currentTime = new Date();
    let expTime = new Date(expiration);

    let timeDifference = expTime.getTime() - currentTime.getTime();

    let totalSeconds = Math.floor(timeDifference / 1000);
    let totalMinutes = Math.floor(totalSeconds / 60);
    let totalHours = Math.floor(totalMinutes / 60);
    let days = Math.floor(totalHours / 24);

    let hours = totalHours % 24;
    let minutes = totalMinutes % 60;
    let seconds = totalSeconds % 60;

    if (minutes === 0 && hours === 0 && days === 0 && seconds <= 0) {
        clearInterval(timeoutInterval);
        return "0 seconds (deleted)";
    } else if (seconds <= 0) {
        seconds = 59;
        minutes -= 1;
    }

    let label = `${seconds} second${seconds > 1 ? "s" : ""}`;

    if (minutes > 0) {
        label = `${minutes} minute${minutes > 1 ? "s" : ""}, ` + label;
    }

    if (hours > 0) {
        label = `${hours} hour${hours > 1 ? "s" : ""}, ` + label;
    }

    if (days > 0) {
        label = `${days} day${days > 1 ? "s" : ""}, ` + label;
    }

    return label;
}

const promptPassword = (download) => {
    let loading = document.getElementById("loading");
    loading.style.display = "none";

    let downloadDiv = document.getElementById("password-prompt-div");
    downloadDiv.style.display = "inherit";

    let password = document.getElementById("password") as HTMLInputElement;
    let btn = document.getElementById("submit") as HTMLButtonElement;

    btn.addEventListener("click", async () => {
        let secret = fromURLSafeBase64(getSecret());

        setFormEnabled(false);
        updatePasswordBtn("Validating", true);

        let [key, _] = await crypto.deriveSendingKey(password.value, secret);

        setFormEnabled(true);

        decryptName(key, download.name).then(decryptedName => {
            showDownload(decryptedName, download, key);
        }).catch(() => {
            updatePasswordBtn("Submit", false);
            alert("Incorrect password");
        });
    });
}

const downloadText = (download, key, callback) => {
    const fetch = () => {
        let xhr = new XMLHttpRequest();
        let url = Endpoints.format(Endpoints.DownloadSendFileData, download.id, "1");
        xhr.open("GET", url, true);
        xhr.responseType = "blob";

        xhr.onreadystatechange = async () => {
            if (xhr.readyState === 4 && xhr.status === 200) {
                let data = new Uint8Array(await xhr.response.arrayBuffer());
                let decryptedChunk = await crypto.decryptChunk(key, data);
                let decryptedText = new TextDecoder().decode(decryptedChunk);
                displayText(decryptedText);
                callback(true);
            } else if (xhr.readyState === 4 && xhr.status !== 200) {
                alert(`Error ${xhr.status}: ${xhr.responseText}`);
                callback(false);
            }
        };

        xhr.send();
    }

    fetch();
}

const displayText = (text) => {
    let plaintextDiv = document.getElementById("plaintext-div");
    let plaintextContent = document.getElementById("plaintext-content");

    plaintextDiv.style.display = "initial";
    plaintextContent.innerText = text;
}

if (document.readyState !== "loading") {
    init();
} else {
    document.addEventListener("DOMContentLoaded", () => {
        init();
    });
}